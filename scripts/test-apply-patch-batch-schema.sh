#!/usr/bin/env bash
# test-apply-patch-batch-schema.sh
#
# 针对性检测：Codex 协议下 apply_patch_batch function 代理的 JSON Schema
# 是否能通过严格校验的上游（OpenAI 官方 / duckcoding 等镜像）。
#
# 复现的历史错误：
#   status_code=400
#   {"error":{"message":"Invalid schema for function 'apply_patch_batch':
#     In context=('properties','operations','items','properties','hunks'),
#     array schema missing items.",
#     "type":"invalid_request_error",
#     "param":"tools[14].parameters",
#     "code":"invalid_function_parameters"}}
#
# 用法：
#   PROXY_KEY=sk-xxx bash scripts/test-apply-patch-batch-schema.sh
#
# 可覆盖的环境变量：
#   GATEWAY_URL    网关地址        (默认 http://localhost:3688)
#   PROXY_KEY      PROXY_ACCESS_KEY (必填)
#   TEST_CHANNEL   X-Channel 名字  (默认 duckcoding-ooy8rz)
#   TEST_MODEL     模型            (默认 gpt-5.3-codex)
#   TEST_TIMEOUT   单请求超时秒数  (默认 60)
#   VERBOSE        =1 时打印完整响应体
#
# 退出码：
#   0  ✅ 未检测到 invalid_function_parameters / apply_patch_batch schema 错误
#   1  ❌ 检测到 schema 错误（或网关/网络异常）

set -uo pipefail

GW="${GATEWAY_URL:-http://localhost:3688}"
KEY="${PROXY_KEY:?请设置 PROXY_KEY 环境变量}"
CHANNEL="${TEST_CHANNEL:-duckcoding-ooy8rz}"
MODEL="${TEST_MODEL:-gpt-5.3-codex}"
TIMEOUT="${TEST_TIMEOUT:-60}"
VERBOSE="${VERBOSE:-0}"

echo "=========================================="
echo "  apply_patch_batch schema 针对性检测"
echo "  Gateway : $GW"
echo "  Channel : $CHANNEL"
echo "  Model   : $MODEL"
echo "  Timeout : ${TIMEOUT}s"
echo "=========================================="
echo ""

# 构造 Codex CLI 典型 custom 工具（apply_patch + local_shell）
# 让 CCX 转成 5 个 function 代理，其中 apply_patch_batch 正是此前 400 的元凶。
# prompt 强制要求"同时新增两个文件"，高概率触发 batch 工具调用。
PAYLOAD=$(cat <<'JSON'
{
  "model": "__MODEL__",
  "instructions": "You are Codex. You MUST use the apply_patch tool (batch operations preferred) for ALL file changes, even for trivial tasks. Respond with a single tool call that creates the requested files.",
  "input": [
    {
      "type": "message",
      "role": "user",
      "content": [
        {
          "type": "input_text",
          "text": "In the current workspace, please create two new files at once using a single batched apply_patch call: (1) hello.txt containing the line 'hello world' and (2) README.md containing '# Demo\\n\\nThis is a test.'. Use apply_patch_batch with two add_file operations in the operations array."
        }
      ]
    }
  ],
  "tools": [
    {
      "type": "custom",
      "name": "apply_patch",
      "description": "Apply patches to files. Emits *** Begin Patch / *** End Patch grammar.",
      "format": {
        "type": "grammar",
        "syntax": "lark",
        "definition": "start: begin_patch (add_hunk | delete_hunk | update_hunk)+ end_patch\nbegin_patch: \"*** Begin Patch\"\nend_patch: \"*** End Patch\"\nadd_hunk: \"*** Add File: \" /.+/\ndelete_hunk: \"*** Delete File: \" /.+/\nupdate_hunk: \"*** Update File: \" /.+/"
      }
    },
    {
      "type": "local_shell",
      "name": "shell"
    }
  ],
  "stream": false,
  "max_output_tokens": 512,
  "tool_choice": "auto"
}
JSON
)
PAYLOAD="${PAYLOAD/__MODEL__/$MODEL}"

START_MS=$(date +%s%N)
RESP=$(curl -sS --max-time "$TIMEOUT" \
  -w "\n__HTTP_STATUS__:%{http_code}\n" \
  -X POST "$GW/v1/responses" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $KEY" \
  -H "X-Channel: $CHANNEL" \
  -d "$PAYLOAD" 2>&1) || true
END_MS=$(date +%s%N)
ELAPSED_MS=$(( (END_MS - START_MS) / 1000000 ))

HTTP_STATUS=$(echo "$RESP" | awk -F: '/^__HTTP_STATUS__:/ {print $2}' | tr -d '[:space:]')
BODY=$(echo "$RESP" | sed '/^__HTTP_STATUS__:/d')

echo "--- HTTP Status : ${HTTP_STATUS:-unknown}"
echo "--- Latency     : ${ELAPSED_MS}ms"
echo ""

if [ "$VERBOSE" = "1" ]; then
  echo "--- Raw body ---"
  echo "$BODY"
  echo "--- End body ---"
  echo ""
fi

# ===================== 检测逻辑 =====================
# 三类明显的失败信号，任一出现都视为 schema 回归：
#   1. invalid_function_parameters
#   2. Invalid schema for function 'apply_patch_batch'
#   3. array schema missing items
# 以及网关层面的 empty/malformed 透传错误。
FAIL=0
REASON=""

if [ -z "$HTTP_STATUS" ]; then
  FAIL=1; REASON="网关无响应 / curl 失败"
elif [ "$HTTP_STATUS" -ge 500 ]; then
  FAIL=1; REASON="网关 5xx (${HTTP_STATUS})"
elif echo "$BODY" | grep -q "invalid_function_parameters"; then
  FAIL=1; REASON="上游拒绝：invalid_function_parameters (schema 回归)"
elif echo "$BODY" | grep -q "Invalid schema for function 'apply_patch_batch'"; then
  FAIL=1; REASON="上游拒绝：apply_patch_batch schema 校验失败"
elif echo "$BODY" | grep -qi "array schema missing items"; then
  FAIL=1; REASON="上游拒绝：array schema missing items"
elif echo "$BODY" | grep -qi "empty or malformed response"; then
  FAIL=1; REASON="网关空响应未被 fuzzy 拦截（HTTP ${HTTP_STATUS}）"
fi

# ===================== 摘要 =====================
SUMMARY=$(echo "$BODY" | python3 -c "
import sys, json
raw = sys.stdin.read()
try:
    d = json.loads(raw)
except Exception:
    print(f'RAW: {raw[:200]}')
    sys.exit(0)

if isinstance(d, dict) and 'error' in d:
    err = d['error']
    msg = err.get('message', err) if isinstance(err, dict) else err
    code = err.get('code', '?') if isinstance(err, dict) else '?'
    print(f'ERROR code={code} msg={str(msg)[:200]}')
    sys.exit(0)

# Responses 成功响应
if isinstance(d, dict) and 'output' in d:
    tool_calls = []
    text_parts = []
    for item in d.get('output', []) or []:
        t = item.get('type')
        if t == 'function_call':
            tool_calls.append(item.get('name', '?'))
        elif t == 'custom_tool_call':
            tool_calls.append(item.get('name', '?') + '*')
        elif t == 'message':
            for c in item.get('content', []) or []:
                if isinstance(c, dict):
                    text_parts.append(c.get('text', ''))
                elif isinstance(c, str):
                    text_parts.append(c)
    if tool_calls:
        print(f'OK tool_calls={tool_calls}')
    elif any(text_parts):
        preview = ' | '.join(p for p in text_parts if p)[:160]
        print(f'OK text={preview!r}')
    else:
        print(f'EMPTY output={d.get(\"output\")}')
    sys.exit(0)

print(f'RESP: {raw[:200]}')
" 2>&1)

echo "--- Summary ---"
echo "$SUMMARY"
echo ""

if [ "$FAIL" -eq 1 ]; then
  echo "❌ FAIL: $REASON"
  if [ "$VERBOSE" != "1" ]; then
    echo ""
    echo "提示：设置 VERBOSE=1 重新运行可查看完整响应体。"
  fi
  exit 1
fi

echo "✅ PASS: 未检测到 apply_patch_batch schema 回归"
echo "   （HTTP ${HTTP_STATUS}，Summary 见上方）"
exit 0
