#!/usr/bin/env python3
"""Generate identifiers for ai-dandelion generated apps."""

from __future__ import annotations

import argparse
import json
import re
import uuid


def slugify(value: str) -> str:
    value = value.strip().lower()
    value = re.sub(r"[^a-z0-9]+", "-", value)
    value = re.sub(r"-+", "-", value).strip("-")
    return value or "business-app"


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--uuid", dest="uuid_value", help="Use a specific UUID instead of generating one.")
    parser.add_argument("--id", dest="row_id", help="Function auto-increment primary key used for data tables.")
    parser.add_argument("--auto-id", help="Deprecated alias for --id.")
    parser.add_argument("--function-id", help="Deprecated alias for --id.")
    parser.add_argument("--entity", action="append", default=[], help="Entity/table suffix for multi-table apps.")
    args = parser.parse_args()

    feature_uuid = (args.uuid_value or str(uuid.uuid4())).lower()
    app_id = feature_uuid
    row_id = args.row_id or args.auto_id or args.function_id
    if not row_id:
        parser.error("--id is required")
    table_prefix = "func_" + re.sub(r"[^0-9a-zA-Z_]+", "_", row_id).strip("_").lower()
    if args.entity:
        tables = [f"{table_prefix}_{slugify(entity).replace('-', '_')}" for entity in args.entity]
    else:
        tables = [table_prefix]
    payload = {
        "uuid": feature_uuid,
        "appId": app_id,
        "folder": f"generated_apps/{app_id}",
        "tablePrefix": table_prefix,
        "tables": tables,
    }
    print(json.dumps(payload, ensure_ascii=False, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
