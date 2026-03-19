# OpenViking 使用指南

## 概述

OpenViking 是字节跳动/火山引擎开源的 AI Agent 上下文数据库，通过类 Unix 文件系统范式管理 AI Agent 的上下文（资源、记忆、技能）。

**核心特性：**
- 文件系统范式：使用 `viking://` URI 统一管理所有上下文
- 三层上下文层级：L0(摘要) → L1(概览) → L2(完整内容)
- 双层存储：AGFS(内容存储) + 向量索引
- 多租户支持：Account(工作区) → User(用户) 隔离

---

## 一、 部署方式

### 1.1 嵌入模式（本地开发）

```python
import openviking as ov

# 初始化客户端（数据存储在本地 ./data 目录）
client = ov.OpenViking(path="./data")
client.initialize()

# 使用完毕后关闭
client.close()
```

### 1.2 服务端模式（生产环境）

```bash
# 启动服务（需要先配置好 ~/.openviking/ov.conf）
python -m openviking serve --port 1933

# 或后台运行
nohup openviking-server > /data/log/openviking.log 2>&1 &
```

**服务端配置文件 (`~/.openviking/ov.conf`) 示例：**

```json
{
  "embedding": {
    "dense": {
      "api_base": "https://ark.cn-beijing.volces.com/api/v3",
      "api_key": "your-embedding-api-key",
      "provider": "volcengine",
      "dimension": 1024,
      "model": "doubao-embedding-vision-250615",
      "input": "multimodal"
    }
  },
  "vlm": {
    "api_base": "https://ark.cn-beijing.volces.com/api/v3",
    "api_key": "your-vlm-api-key",
    "provider": "volcengine",
    "max_retries": 2,
    "model": "doubao-seed-1-8-251228"
  }
}
```

### 1.3 客户端连接服务端

**Python SDK:**

```python
import openviking as ov

# 无认证连接
client = ov.SyncHTTPClient(url="http://localhost:1933")

# 有认证连接
client = ov.SyncHTTPClient(url="http://localhost:1933", api_key="your-user-key")

try:
    client.initialize()
    # ... 使用 client ...
finally:
    client.close()
```

**CLI 配置 (`~/.openviking/ovcli.conf`):**

```json
{
  "url": "http://localhost:1933",
  "api_key": "your-user-key"
}
```

```bash
# 验证连接
openviking observer system

# 基本操作
openviking ls viking://resources/
openviking find "what is openviking"
```

**HTTP API (curl):**

```bash
# 健康检查
curl http://localhost:1933/health

# 需要 API Key 的请求
curl -X GET "http://localhost:1933/api/v1/fs/ls?uri=viking://resources/" \
  -H "X-API-Key: your-user-key"
```

---

## 二、 多用户访问 URL（多租户）

### 2.1 核心概念

OpenViking 通过 **Account(工作区)** + **User(用户)** 实现多租户隔离：

**角色权限：**

| 角色 | 说明 |
|------|------|
| ROOT | 系统管理员，拥有全部权限 |
| ADMIN | 工作区管理员，管理本 account 内的用户 |
| USER | 普通用户 |

### 2.2 创建工作区和用户

```bash
# 步骤 1：ROOT 创建工作区，指定 alice 为首个 admin
openviking admin create-account acme --admin alice
# 返回：{"user_key": "7f3a9c1e..."}  ← alice 的 API Key

# 步骤 2：alice（admin）注册普通用户 bob
openviking admin register-user acme bob --role user
# 返回：{"user_key": "d91f5b2a..."}  ← bob 的 API Key
```

**HTTP API 等效：**

```bash
# 创建工作区
curl -X POST http://localhost:1933/api/v1/admin/accounts \
  -H "Content-Type: application/json" \
  -H "X-API-Key: <ROOT_KEY>" \
  -d '{"account_id": "acme", "admin_user_id": "alice"}'

# 注册用户（使用 admin 的 key）
curl -X POST http://localhost:1933/api/v1/admin/accounts/acme/users \
  -H "Content-Type: application/json" \
  -H "X-API-Key: <ALICE_KEY>" \
  -d '{"user_id": "bob", "role": "user"}'
```

### 2.3 用户访问 URL

每个用户使用自己的 API Key 访问服务：

```bash
# alice 访问
curl -X GET "http://localhost:1933/api/v1/fs/ls?uri=viking://resources/" \
  -H "X-API-Key: 7f3a9c1e..."

# bob 访问
curl -X GET "http://localhost:1933/api/v1/fs/ls?uri=viking://resources/" \
  -H "X-API-Key: d91f5b2a..."
```

### 2.4 管理命令汇总

```bash
# 列出所有工作区（ROOT）
openviking admin list-accounts

# 列出工作区内用户
openviking admin list-users acme

# 修改用户角色（ROOT）
openviking admin set-role acme bob admin

# 重新生成用户 Key（旧 Key 立即失效）
openviking admin regenerate-key acme bob

# 移除用户
openviking admin remove-user acme bob

# 删除工作区（ROOT）
openviking admin delete-account acme
```

---

## 三、 URI 体系

### 3.1 URI 结构

```
viking://<scope>/<path>

scope: resources | user | agent | session
```

**三种上下文作用域：**

| 作用域 | URI 前缀 | 说明 |
|--------|----------|------|
| resources | `viking://resources/` | 项目资源、文档、代码库 |
| user | `viking://user/{user_id}/` | 用户记忆（每人独立） |
| agent | `viking://agent/` | Agent 技能、记忆、指令 |
| session | `viking://session/{session_id}/` | 会话临时存储 |

### 3.2 用户记忆存储位置

```
viking://user/{user_id}/
├── memories/
│   ├── .overview.md          # L1: 用户个人信息 (profile)
│   ├── preferences/          # 用户偏好
│   │   └── theme.md
│   ├── entities/             # 重要实体（人物、项目等）
│   │   └── project-xyz.md
│   └── events/               # 重要事件
│       └── 2024-01-15-meeting.md
```

---

## 四、 存储记忆

### 4.1 通过会话自动提取记忆

**完整流程：**

```python
import openviking as ov
from openviking.message import TextPart

client = ov.SyncHTTPClient(url="http://localhost:1933", api_key="your-user-key")
client.initialize()

# 1. 创建会话
session = client.session()

# 2. 添加对话消息
session.add_message("user", [
    TextPart(text="我叫张三，我喜欢使用暗色主题")
])
session.add_message("assistant", [
    TextPart(text="好的，我记住了您的偏好。")
])

# 3. 提交会话（自动提取记忆到 viking://user/{user_id}/memories/）
result = session.commit()
print(f"提取的记忆数: {result['memories_extracted']}")

client.close()
```

**HTTP API 等效：**

```bash
# 1. 创建会话
curl -X POST http://localhost:1933/api/v1/sessions \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-user-key"
# 返回：{"session_id": "abc123"}

# 2. 添加消息
curl -X POST http://localhost:1933/api/v1/sessions/abc123/messages \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-user-key" \
  -d '{"role": "user", "content": "我叫张三，我喜欢使用暗色主题"}'

# 3. 提交会话
curl -X POST http://localhost:1933/api/v1/sessions/abc123/commit \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-user-key"
```

### 4.2 记忆分类

会话提交后，系统自动将记忆提取到以下位置：

| 分类 | 位置 | 说明 |
|------|------|------|
| profile | `user/memories/.overview.md` | 用户个人信息 |
| preferences | `user/memories/preferences/` | 按主题分类的用户偏好 |
| entities | `user/memories/entities/` | 重要实体（人物、项目等） |
| events | `user/memories/events/` | 重要事件 |
| cases | `agent/memories/cases/` | 问题-解决方案案例 |
| patterns | `agent/memories/patterns/` | 交互模式 |

### 4.3 直接写入记忆

```python
# 手动创建记忆文件
client.mkdir("viking://user/alice/memories/preferences/")

# 写入偏好内容
client.write("viking://user/alice/memories/preferences/theme.md", """
# 主题偏好

用户喜欢暗色主题，偏好简洁的界面风格。

- 深色模式: 是
- 字体大小: 中等
- 语言: 中文
""")
```

---

## 五、 查询记忆

### 5.1 语义搜索（find）

```python
# 基本搜索
results = client.find("用户的主题偏好")

for ctx in results.resources:
    print(f"URI: {ctx.uri}")
    print(f"分数: {ctx.score:.3f}")
    print(f"摘要: {ctx.abstract}")

# 限定搜索范围
# 仅在用户记忆中搜索
results = client.find(
    "偏好设置",
    target_uri="viking://user/alice/memories/"
)

# 使用分数阈值
results = client.find(
    "主题",
    target_uri="viking://user/",
    score_threshold=0.5,
    limit=5
)
```

**HTTP API:**

```bash
# 基本搜索
curl -X POST http://localhost:1933/api/v1/search/find \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-user-key" \
  -d '{"query": "用户的主题偏好", "limit": 10}'

# 限定范围搜索
curl -X POST http://localhost:1933/api/v1/search/find \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-user-key" \
  -d '{
    "query": "偏好设置",
    "target_uri": "viking://user/alice/memories/",
    "score_threshold": 0.5
  }'
```

### 5.2 上下文感知搜索（search）

`search` 支持会话上下文，能理解对话意图。

```python
from openviking.message import TextPart

# 创建带对话上下文的会话
session = client.session()
session.add_message("user", [
    TextPart(text="我在配置用户界面")
])
session.add_message("assistant", [
    TextPart(text="我可以帮您查找相关的偏好设置。")
])

# 搜索能够理解"设置"指的是"界面设置"的上下文
results = client.search("相关配置", session=session)

for ctx in results.resources:
    print(f"找到: {ctx.uri}")
    print(f"匹配原因: {ctx.match_reason}")
```

**HTTP API:**

```bash
# 带会话上下文的搜索
curl -X POST http://localhost:1933/api/v1/search/search \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-user-key" \
  -d '{
    "query": "相关配置",
    "session_id": "abc123",
    "limit": 10
  }'
```

### 5.3 正则搜索（grep）

```python
# 按模式搜索内容
results = client.grep(
    "viking://user/alice/memories/",
    "主题|偏好",
    case_insensitive=True
)

print(f"找到 {results['count']} 处匹配")
for match in results['matches']:
    print(f"  {match['uri']}:{match['line']}")
    print(f"  {match['content']}")
```

**HTTP API:**

```bash
curl -X POST http://localhost:1933/api/v1/search/grep \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-user-key" \
  -d '{
    "uri": "viking://user/alice/memories/",
    "pattern": "主题|偏好",
    "case_insensitive": true
  }'
```

### 5.4 文件模式匹配（glob）

```python
# 查找所有 markdown 文件
results = client.glob("**/*.md", "viking://user/alice/memories/")

print(f"找到 {results['count']} 个文件:")
for uri in results['matches']:
    print(f"  {uri}")
```

---

## 六、 读取内容

### 6.1 三层内容读取

```python
# L0: 摘要（约 100 token）
abstract = client.abstract("viking://user/alice/memories/preferences/")
print(f"摘要: {abstract}")

# L1: 概览（约 2k token，适用于目录）
overview = client.overview("viking://user/alice/memories/preferences/")
print(f"概览: {overview}")

# L2: 完整内容
content = client.read("viking://user/alice/memories/preferences/theme.md")
print(f"内容: {content}")
```

**HTTP API:**

```bash
# L0 摘要
curl -X GET "http://localhost:1933/api/v1/content/abstract?uri=viking://user/alice/memories/" \
  -H "X-API-Key: your-user-key"

# L1 概览
curl -X GET "http://localhost:1933/api/v1/content/overview?uri=viking://user/alice/memories/" \
  -H "X-API-Key: your-user-key"

# L2 完整内容
curl -X GET "http://localhost:1933/api/v1/content/read?uri=viking://user/alice/memories/preferences/theme.md" \
  -H "X-API-Key: your-user-key"
```

### 6.2 目录操作

```python
# 列出目录
entries = client.ls("viking://user/alice/memories/")
for entry in entries:
    type_str = "目录" if entry['isDir'] else "文件"
    print(f"{entry['name']} - {type_str}")

# 递归列出
entries = client.ls("viking://user/alice/", recursive=True)

# 获取目录树
tree = client.tree("viking://user/alice/memories/")
for entry in tree:
    print(f"{entry['rel_path']}")
```

---

## 七、 完整示例

### 7.1 多用户对话系统

```python
import openviking as ov
from openviking.message import TextPart

class UserMemoryService:
    def __init__(self, server_url: str):
        self.server_url = server_url

    def get_client(self, user_key: str):
        """获取用户的客户端"""
        return ov.SyncHTTPClient(url=self.server_url, api_key=user_key)

    def chat(self, user_key: str, user_id: str, message: str) -> str:
        """处理用户消息并返回响应"""
        client = self.get_client(user_key)
        try:
            client.initialize()

            # 1. 创建会话
            session = client.session()

            # 2. 搜索用户相关记忆
            memories = client.find(
                f"用户{user_id}的偏好和历史",
                target_uri=f"viking://user/{user_id}/memories/"
            )

            # 构建上下文
            context = ""
            if memories.resources:
                context = f"已知用户信息：{memories.resources[0].abstract}"

            # 3. 添加用户消息
            session.add_message("user", [TextPart(text=message)])

            # 4. 模拟 AI 响应（实际应调用 LLM）
            response = f"基于您的偏好，我建议... (上下文: {context[:100]}...)"

            # 5. 添加助手响应
            session.add_message("assistant", [TextPart(text=response)])

            # 6. 提交会话（自动提取记忆）
            result = session.commit()

            return response
        finally:
            client.close()

# 使用示例
service = UserMemoryService("http://localhost:1933")

# 用户 alice 的对话
response = service.chat(
    user_key="alice-api-key",
    user_id="alice",
    message="我想要一个简洁的深色界面"
)
```

### 7.2 管理脚本

```bash
#!/bin/bash
# openviking-admin.sh

SERVER="http://localhost:1933"
ROOT_KEY="your-root-key"

# 创建新工作区
create_workspace() {
    local account_id=$1
    local admin_id=$2

    result=$(curl -s -X POST "$SERVER/api/v1/admin/accounts" \
        -H "Content-Type: application/json" \
        -H "X-API-Key: $ROOT_KEY" \
        -d "{\"account_id\": \"$account_id\", \"admin_user_id\": \"$admin_id\"}")

    echo "$result"
}

# 注册用户
register_user() {
    local account_id=$1
    local user_id=$2
    local admin_key=$3

    result=$(curl -s -X POST "$SERVER/api/v1/admin/accounts/$account_id/users" \
        -H "Content-Type: application/json" \
        -H "X-API-Key: $admin_key" \
        -d "{\"user_id\": \"$user_id\", \"role\": \"user\"}")

    echo "$result"
}

# 搜索用户记忆
search_memories() {
    local user_key=$1
    local query=$2

    curl -s -X POST "$SERVER/api/v1/search/find" \
        -H "Content-Type: application/json" \
        -H "X-API-Key: $user_key" \
        -d "{\"query\": \"$query\", \"target_uri\": \"viking://user/\"}"
}

# 示例使用
# create_workspace "myapp" "admin1"
# register_user "myapp" "user1" "<admin_key>"
# search_memories "<user_key>" "用户偏好"
```

---

## 八、 CLI 命令速查

```bash
# === 系统管理 ===
openviking observer system              # 查看系统状态

# === 资源管理 ===
openviking add-resource <path_or_url>   # 添加资源
openviking ls viking://resources/       # 列出资源
openviking rm viking://resources/xxx/   # 删除资源

# === 搜索 ===
openviking find "query"                 # 语义搜索
openviking search "query" --session-id <id>  # 带上下文搜索
openviking grep viking:// "pattern"     # 正则搜索
openviking glob "**/*.md"               # 文件匹配

# === 会话管理 ===
openviking session new                   # 创建会话
openviking session list                  # 列出会话
openviking session get <id>              # 获取会话
openviking session commit <id>           # 提交会话

# === 多租户管理 ===
openviking admin create-account <id> --admin <user>
openviking admin list-accounts
openviking admin list-users <account>
openviking admin register-user <account> <user> --role user
openviking admin remove-user <account> <user>
openviking admin set-role <account> <user> admin
openviking admin regenerate-key <account> <user>
openviking admin delete-account <account>
```

---

## 九、 最佳实践

### 9.1 定期提交会话

```python
# 在重要交互后提交
if len(session.messages) > 10:
    session.commit()
```

### 9.2 跟踪实际使用的内容

```python
# 仅标记实际有帮助的上下文
if context_was_useful:
    session.used(contexts=[ctx.uri])
```

### 9.3 使用会话上下文进行搜索

```python
# 结合对话上下文可获得更好的搜索结果
results = client.search(query, session=session)
```

### 9.4 渐进式读取内容

```python
# 先从 L0 摘要开始，按需加载更多
results = client.find("query")
for ctx in results.resources:
    # L0 已包含在 ctx.abstract 中
    if need_more_detail:
        if not ctx.is_leaf:
            # 目录：获取 L1 概览
            overview = client.overview(ctx.uri)
        else:
            # 文件：加载 L2 完整内容
            content = client.read(ctx.uri)
```

---

## 十、 常见问题

### Q: 如何实现用户数据隔离？

A: 每个用户使用独立的 API Key，数据通过 URI 路径隔离：
- 用户 A: `viking://user/alice/memories/`
- 用户 B: `viking://user/bob/memories/`

### Q: 记忆是如何提取的？

A: 会话 `commit()` 时，系统使用 VLM 自动分析对话内容，提取 6 类记忆：
- profile（个人信息）
- preferences（偏好）
- entities（实体）
- events（事件）
- cases（案例）
- patterns（模式）

### Q: 如何批量导入历史数据？

A: 使用 `add_resource` API 添加本地文件或 URL：

```bash
openviking add-resource ./docs/
openviking add-resource https://example.com/doc.html
```

### Q: 服务端如何持久化数据？

A: 数据默认存储在服务启动目录的 `./data` 目录。可通过配置文件指定其他路径。
