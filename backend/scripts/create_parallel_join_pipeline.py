#!/usr/bin/env python3
"""通过 API 创建「并行分支 + 聚合节点」示例流水线，并启动一次实例验证。"""

from __future__ import annotations

import json
import sys
import urllib.error
import urllib.request

BASE_URL = "http://localhost:8080/api/v1"


def request(method: str, path: str, body: dict | None = None) -> dict:
    data = None
    headers = {"Content-Type": "application/json"}
    if body is not None:
        data = json.dumps(body).encode()
    req = urllib.request.Request(f"{BASE_URL}{path}", data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req) as resp:
            raw = resp.read().decode()
            return json.loads(raw) if raw else {}
    except urllib.error.HTTPError as e:
        detail = e.read().decode()
        raise RuntimeError(f"{method} {path} failed ({e.code}): {detail}") from e


def main() -> None:
    print("==> 1. 创建流水线模板")
    tpl = request("POST", "/flow-templates", {"name": "parallel-join-demo"})
    tpl_id = tpl["id"]
    print(f"    template_id={tpl_id}")

    print("==> 2. 获取草稿版本 ID")
    versions = request("GET", f"/flow-templates/{tpl_id}/versions")
    version_id = versions["items"][0]["id"]
    print(f"    version_id={version_id}")

    base = f"/flow-templates/{tpl_id}/versions/{version_id}"

    print("==> 3. 添加入参 a, b, c")
    for name, desc in [("a", "控制节点A"), ("b", "控制节点B"), ("c", "控制节点C")]:
        request(
            "POST",
            f"{base}/params",
            {
                "name": name,
                "type": "string",
                "default_value": "1" if name != "c" else "0",
                "required": False,
                "description": desc,
            },
        )

    print("==> 4. 添加节点")
    start_id = request("POST", f"{base}/nodes", {"type": "start", "name": "start"})["id"]
    print_id = request(
        "POST",
        f"{base}/nodes",
        {"type": "task", "name": "print_abc", "script": 'echo "a=$a b=$b c=$c"'},
    )["id"]
    node_a_id = request(
        "POST",
        f"{base}/nodes",
        {"type": "task", "name": "node_a", "script": 'echo "d=1" >> "$FLOW_ENV_OUT"'},
    )["id"]
    node_b_id = request(
        "POST",
        f"{base}/nodes",
        {"type": "task", "name": "node_b", "script": 'echo "e=1" >> "$FLOW_ENV_OUT"'},
    )["id"]
    node_c_id = request(
        "POST",
        f"{base}/nodes",
        {"type": "task", "name": "node_c", "script": 'echo "f=1" >> "$FLOW_ENV_OUT"'},
    )["id"]
    join_id = request(
        "POST",
        f"{base}/nodes",
        {"type": "join", "name": "join_def", "script": 'echo "d=$d e=$e f=$f"'},
    )["id"]
    end_id = request("POST", f"{base}/nodes", {"type": "end", "name": "end"})["id"]
    print(
        f"    start={start_id} print={print_id} A={node_a_id} B={node_b_id} "
        f"C={node_c_id} join={join_id} end={end_id}"
    )

    print("==> 5. 添加边")
    request(
        "POST",
        f"{base}/nodes",
        {
            "type": "edge",
            "name": "start_to_print",
            "from_node_id": start_id,
            "to_node_id": print_id,
            "is_default": True,
        },
    )
    request(
        "POST",
        f"{base}/nodes",
        {
            "type": "edge",
            "name": "print_to_a",
            "from_node_id": print_id,
            "to_node_id": node_a_id,
            "condition": '[ "$a" = "1" ]',
        },
    )
    request(
        "POST",
        f"{base}/nodes",
        {
            "type": "edge",
            "name": "print_to_b",
            "from_node_id": print_id,
            "to_node_id": node_b_id,
            "condition": '[ "$b" = "1" ]',
        },
    )
    request(
        "POST",
        f"{base}/nodes",
        {
            "type": "edge",
            "name": "print_to_c",
            "from_node_id": print_id,
            "to_node_id": node_c_id,
            "condition": '[ "$c" = "1" ]',
        },
    )
    edge_a_join = request(
        "POST",
        f"{base}/nodes",
        {
            "type": "edge",
            "name": "a_to_join",
            "from_node_id": node_a_id,
            "to_node_id": join_id,
            "is_default": True,
        },
    )["id"]
    edge_b_join = request(
        "POST",
        f"{base}/nodes",
        {
            "type": "edge",
            "name": "b_to_join",
            "from_node_id": node_b_id,
            "to_node_id": join_id,
            "is_default": True,
        },
    )["id"]
    request(
        "POST",
        f"{base}/nodes",
        {
            "type": "edge",
            "name": "c_to_join",
            "from_node_id": node_c_id,
            "to_node_id": join_id,
            "is_default": True,
        },
    )
    request(
        "POST",
        f"{base}/nodes",
        {
            "type": "edge",
            "name": "join_to_end",
            "from_node_id": join_id,
            "to_node_id": end_id,
            "is_default": True,
        },
    )
    print(f"    edge_a_to_join={edge_a_join} edge_b_to_join={edge_b_join}")

    print("==> 6. 设置聚合节点前置边条件")
    request(
        "PUT",
        f"{base}/nodes/{join_id}",
        {"required_edge_ids": [edge_a_join, edge_b_join]},
    )

    print("==> 7. 配置手动触发")
    request(
        "PUT",
        f"/flow-templates/{tpl_id}/trigger",
        {"manual": True, "webhook": False, "cron": False, "require_params": False},
    )

    print("==> 8. 发布模板")
    request("POST", f"/flow-templates/{tpl_id}/publish", {})

    print("==> 9. 启动实例 (a=1, b=1, c=0)")
    inst = request(
        "POST",
        f"/flow-templates/{tpl_id}/instances",
        {"trigger_type": "manual", "params": {"a": "1", "b": "1", "c": "0"}},
    )

    env = inst.get("env", {})
    print()
    print("========== 创建完成 ==========")
    print(f"模板 ID:     {tpl_id}")
    print(f"版本 ID:     {version_id}")
    print(f"实例 ID:     {inst['id']}")
    print(f"实例状态:    {inst['status']}")
    print(f"全局 Env:    d={env.get('d', '')}  e={env.get('e', '')}  f={env.get('f', '<unset>')}")
    print()
    print(f"查看详情: curl {BASE_URL}/flow-templates/{tpl_id}/instances/{inst['id']} | python3 -m json.tool")


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:  # noqa: BLE001
        print(f"ERROR: {exc}", file=sys.stderr)
        sys.exit(1)
