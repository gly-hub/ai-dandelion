#!/usr/bin/env bash

if [ -z "${BASH_VERSION:-}" ]; then
  exec bash "$0" "$@"
fi

SCRIPT_DIR="$(cd -- "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TMP_INCLUDE_ROOT=""
PROTO_INCLUDE_ROOT=""
GOGO_OUT_OPTS=""

cleanup() {
  if [ -n "$TMP_INCLUDE_ROOT" ] && [ -d "$TMP_INCLUDE_ROOT" ]; then
    rm -rf "$TMP_INCLUDE_ROOT"
  fi
}

trap cleanup EXIT

cd "$SCRIPT_DIR" || exit 1
shopt -s nullglob

# 一键 gogo proto文件生成grpc代码
# 使用方法:
#   1. 全量生成(原有行为):         bash proto/gogo.sh
#   2. 生成指定目录所有 proto:     bash proto/gogo.sh ai-agent
#   3. 生成指定目录下指定 proto:   bash proto/gogo.sh ai-agent ai-agent-service.proto
#   4. 直接传 proto 文件路径:      bash proto/gogo.sh ai-agent/ai-agent-service.proto
#      多个文件(支持跨目录):        bash proto/gogo.sh ai-agent/ai-agent-service.proto lib/any.proto
#      参数也兼容带 proto/ 前缀写法,如 proto/ai-agent/ai-agent-service.proto

require_command() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "缺少依赖命令: $cmd"
    exit 1
  fi
}

normalize_path() {
  local path="${1%/}"
  if [ -e "$path" ]; then
    printf '%s\n' "$path"
    return 0
  fi

  if [[ "$path" == proto/* ]] && [ -e "${path#proto/}" ]; then
    printf '%s\n' "${path#proto/}"
    return 0
  fi

  printf '%s\n' "$path"
}

link_module_include() {
  local module_path="$1"
  local required_proto="$2"
  local module_dir
  local parent_dir

  module_dir="$(go list -m -f '{{.Dir}}' "$module_path" 2>/dev/null)" || return 1
  if [ ! -f "$module_dir/$required_proto" ]; then
    return 1
  fi

  parent_dir="$TMP_INCLUDE_ROOT/$(dirname "$module_path")"
  mkdir -p "$parent_dir" || return 1
  ln -sfn "$module_dir" "$TMP_INCLUDE_ROOT/$module_path" || return 1
}

prepare_proto_include_root() {
  TMP_INCLUDE_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/ai-dandelion-proto.XXXXXX")" || return 1

  link_module_include "github.com/gogo/protobuf" "gogoproto/gogo.proto" || return 1
  link_module_include "github.com/team-dandelion/quickgo" "grpcep/lib.proto" || return 1

  printf '%s\n' "$TMP_INCLUDE_ROOT"
}

build_gogo_out_opts() {
  local quickgo_dir
  quickgo_dir="$(go list -m -f '{{.Dir}}' github.com/team-dandelion/quickgo 2>/dev/null)" || return 1

  if [ ! -f "$quickgo_dir/grpcep/lib.proto" ]; then
    return 1
  fi

  # quickgo/grpcep/lib.proto 的 go_package 是相对路径，需显式映射为可导入包路径。
  printf '%s\n' "Mgithub.com/team-dandelion/quickgo/grpcep/lib.proto=github.com/team-dandelion/quickgo/grpcep,plugins=grpc"
}

# gen_dir <dir> [proto_file ...]
# 生成指定目录(可选指定部分 proto 文件)的 grpc 代码
gen_dir() {
  local dir="$1"
  shift
  local baseName
  baseName=$(basename "$dir")
  if [ "$baseName" = "protodesc" ]; then
    return
  fi

  local files=()
  if [ $# -eq 0 ]; then
    # 目录下全部 proto
    files=("$dir"/*.proto)
    if [ ${#files[@]} -eq 0 ]; then
      echo "跳过 $dir: 没有找到 proto 文件"
      return 0
    fi
    echo "正在生成 $dir 目录下的 gogo proto 文件"
  else
    # 指定 proto 文件,支持裸文件名或带目录前缀
    local f
    for f in "$@"; do
      if [[ "$f" == "$dir"/* || "$f" == ./"$dir"/* ]]; then
        files+=("$f")
      else
        files+=("$dir/$f")
      fi
    done
    # 校验文件是否存在
    local m
    for m in "${files[@]}"; do
      if [ ! -f "$m" ]; then
        echo "proto 文件 $m 不存在"
        return 1
      fi
    done
    echo "正在生成 $dir 目录下指定 proto 文件: ${files[*]}"
  fi

  protoc -I="$dir" -I=. -I="$PROTO_INCLUDE_ROOT" --gogofaster_out="$GOGO_OUT_OPTS":"$dir" "${files[@]}"
  if [ $? -eq 0 ]; then
    echo "$dir 目录下 gogo proto 生成成功"
  else
    echo "$dir 目录下 gogo proto 生成失败"
    return 1
  fi
}

require_command go
require_command protoc
require_command protoc-gen-gogofaster

PROTO_INCLUDE_ROOT="$(prepare_proto_include_root)" || {
  echo "未找到依赖 proto 文件，请先确认 go.mod replace 路径有效并执行 go mod download"
  exit 1
}

GOGO_OUT_OPTS="$(build_gogo_out_opts)" || {
  echo "构建 gogofaster 参数失败: 未找到 quickgo/grpcep/lib.proto"
  exit 1
}

normalized_args=()
for arg in "$@"; do
  normalized_args+=("$(normalize_path "$arg")")
done
set -- "${normalized_args[@]}"

if [ $# -eq 0 ]; then
  # 无参数: 全量生成
  for dir in *; do
    if [ -d "$dir" ]; then
      gen_dir "$dir"
    fi
  done
else
  first="${1%/}"
  if [ -d "$first" ]; then
    # 目录模式: $1 = 目录, $2... = 可选的指定 proto 文件(不传则生成该目录全部)
    shift
      gen_dir "$first" "$@"
  else
    # 文件模式: 所有参数都是 proto 文件路径, 按目录分组后批量生成
    # (避免用 bash 4 关联数组,macOS 自带 bash 3.2 不支持)
    dirs=""
    for f in "$@"; do
      if [ ! -f "$f" ]; then
        echo "proto 文件 $f 不存在(也不是有效目录)"
        exit 1
      fi
      d=$(dirname "$f")
      case " $dirs " in
        *" $d "*) ;;
        *) dirs="$dirs $d" ;;
      esac
    done
    for d in $dirs; do
      group=()
      for f in "$@"; do
        if [ "$(dirname "$f")" = "$d" ]; then
          group+=("$f")
        fi
      done
      gen_dir "$d" "${group[@]}" || exit 1
    done
  fi
fi
