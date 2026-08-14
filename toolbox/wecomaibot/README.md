# WeCom AI Bot Go Client

This package is the Go implementation of the local `wecom-aibot-python-sdk`
behavior used by ai-agent.

Implemented protocol behavior:

- connect to `wss://openws.work.weixin.qq.com`
- send auth frame with `cmd=aibot_subscribe`
- send heartbeat frame with `cmd=ping`
- dispatch `aibot_msg_callback` as `message.<msgtype>`
- dispatch `aibot_event_callback` as `event.<eventtype>`
- reply stream messages with `cmd=aibot_respond_msg`
- reply welcome messages with `cmd=aibot_respond_welcome_msg`

The ai-agent runtime starts this client directly from enabled Smart Bot channel
configs, so no Python bridge process is required.
