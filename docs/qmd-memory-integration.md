# QMD 记忆处理机制调研与集成指南

## 概述

本文档基于对 [qmd](https://github.com/tobi/qmd) 源码的深入分析，总结其记忆处理的核心思路，并提供将其集成到 nanobot-go 项目的具体指导。

---

## 一、QMD 核心记忆处理思路

### 1.1 整体架构

QMD 采用了**混合搜索 + 多层索引**的记忆架构：

```
┌─────────────────────────────────────────────────────────────────┐
│                    QMD 记忆架构                         │
├─────────────────────────────────────────────────────────────────┤
│                                                          │
│  文档输入 → 分块 → BM25索引 + 向量索引              │
│                  ↓                                         │
│            SQLite 存储 (单库)                              │
│                  ↓                                         │
│         查询扩展 → RRF融合 → LLM重排                  │
│                  ↓                                         │
│            相关性评分 → Top-K 返回                          │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 数据存储设计

#### SQLite 单库设计

QMD 使用单个 SQLite 数据库存储所有数据，位于 `~/.cache/qmd/index.sqlite`：

```sql
-- 文档主表
documents (
  id INTEGER PRIMARY KEY,
  collection TEXT NOT NULL,      -- 所属集合
  path TEXT NOT NULL,              -- 文件路径
  title TEXT NOT NULL,             -- 提取的标题
  hash TEXT NOT NULL,             -- 内容哈希（6字符）
  created_at TEXT,                 -- 创建时间
  modified_at TEXT,                -- 修改时间
  active INTEGER DEFAULT 1,         -- 激活标记
  FOREIGN KEY (hash) REFERENCES content(hash) ON DELETE CASCADE
)

-- 内容表（去重）
content (
  hash TEXT PRIMARY KEY,           -- SHA256 哈希
  doc TEXT NOT NULL,               -- 原始内容
  created_at TEXT NOT NULL
)

-- 全文搜索索引 (BM25)
documents_fts VIRTUAL TABLE USING fts5(
  filepath, title, body,
  tokenize='porter unicode61'
)

-- 向量分块表
content_vectors (
  hash TEXT NOT NULL,             -- 文档哈希
  seq INTEGER NOT NULL,           -- 分块序号 (0,1,2...)
  pos INTEGER NOT NULL,           -- 原文位置
  model TEXT NOT NULL,            -- 嵌入模型
  embedded_at TEXT NOT NULL,      -- 嵌入时间
  PRIMARY KEY (hash, seq)
)

-- 向量索引 (sqlite-vec)
vectors_vec VIRTUAL TABLE USING vec0(
  hash_seq TEXT PRIMARY KEY,        -- "hash_seq" 组合键
  embedding float[768] distance_metric=cosine  -- 向量 + 余弦距离
)

-- 上下文表
path_contexts (
  collection TEXT,
  path TEXT,
  context TEXT,                  -- 路径的描述性上下文
  PRIMARY KEY (collection, path)
)

-- LLM 缓存表
llm_cache (
  hash TEXT PRIMARY KEY,           -- 输入哈希
  result TEXT NOT NULL,           -- 缓存结果
  created_at TEXT NOT NULL
)
```

**关键设计要点：**

1. **内容去重**：`content` 表通过 `hash` 去重，相同内容只存储一次
2. **索引同步**：通过 trigger 自动维护 FTS 索引与 documents 表同步
3. **分块存储**：每篇文档被分成多个 chunk，每个 chunk 独立向量
4. **组合主键**：vectors_vec 使用 `hash_seq` 组合键，确保每个分块唯一

### 1.3 智能分块算法

QMD 的核心创新之一是**基于自然边界的智能分块**：

```typescript
// 分块配置
const CHUNK_SIZE_TOKENS = 900      // 每块约 900 tokens
const CHUNK_OVERLAP_TOKENS = 135   // 15% 重叠
const CHUNK_WINDOW_TOKENS = 200    // 搜索窗口

// 断点评分表
BREAK_PATTERNS = [
  [/\n#{1}(?!#)/g, 100, 'h1'],      // H1 标题
  [/\n#{2}(?!#)/g, 90, 'h2'],       // H2 标题
  [/\n#{3}(?!#)/g, 80, 'h3'],       // H3 标题
  [/\n```/g, 80, 'codeblock'],        // 代码块边界
  [/\n(?:---|\*\*\*|___)\s*\n/g, 60, 'hr'],  // 分隔线
  [/\n\n+/g, 20, 'blank'],            // 空行
  [/\n[-*]\s/g, 5, 'list'],             // 列表项
  [/\n\d+\.\s/g, 5, 'numlist'],         // 有序列表
  [/\n/g, 1, 'newline'],                // 普通换行
]
```

**分块算法流程：**

1. 扫描文档，收集所有潜在断点及其评分
2. 当接近目标大小时，在窗口内搜索最佳断点
3. 使用距离衰减函数：`finalScore = baseScore × (1 - (distance/window)² × 0.7)`
4. 选择得分最高的断点
5. **代码块保护**：断点在代码块内时被忽略

**示例：**

```
文档内容: "# Introduction\n\nThis is the first paragraph.\n## API Reference\n\n..."

分块结果:
- Chunk 0: "# Introduction\n\nThis is the first paragraph.\n"   (在 ## 处断开)
- Chunk 1: "## API Reference\n\n..."                    (继续到下一个 H2 或代码块)
```

### 1.4 混合搜索流程

QMD 的查询采用**四阶段混合搜索**：

```
┌─────────────────────────────────────────────────────────────┐
│                QMD 混合搜索 Pipeline                    │
└─────────────────────────────────────────────────────────────┘

用户查询
    │
    ├───► LLM 查询扩展 ───► [原始查询, 变体1, 变体2]
    │                                         │
    │           ┌───────────────────────────┼───────────────────────┐
    │           ▼                           ▼                       ▼
    │    原始查询 (×2权重)         扩展1                   扩展2
    │    ┌──────────┐              ┌──────────┐           ┌──────────┐
    │    │   FTS    │              │   FTS    │           │   FTS    │
    │    │ (BM25)   │              │ (BM25)   │           │ (BM25)   │
    │    └────┬─────┘              └────┬─────┘           └────┬─────┘
    │         │                          │                       │
    │    原始查询         扩展1            扩展2
    │    ┌──────────┐        ┌──────────┐         ┌──────────┐
    │    │  Vector  │        │  Vector  │         │  Vector  │
    │    │ (语义)   │        │  (语义)   │         │  (语义)   │
    │    └────┬─────┘        └────┬─────┘         └────┬─────┘
    │         │                     │                      │
    │         └─────────────────────┼──────────────────────┘
    │                               │
    │         RRF Fusion (k=60)
    │         - 原始查询 ×2 权重
    │         - Top-rank 奖励 (+0.05 for #1, +0.02 for #2-3)
    │         - 取 Top 30 候选
    │                               │
    │                         Top 30 候选
    │                               │
    │                               ▼
    │                    LLM 重排 (Cross-encoder)
    │                    - 每个候选打分 0-10
    │                    - 返回带 logprobs 的 yes/no
    │                               │
    │         位置感知融合
    │         - Rank 1-3: 75% RRF + 25% 重排
    │         - Rank 4-10: 60% RRF + 40% 重排
    │         - Rank 11+: 40% RRF + 60% 重排
    │                               │
    │                         最终结果 (Top-N)
```

#### 1.4.1 查询扩展

使用微调的 LLM 模型生成查询变体：

```typescript
// LLM 输出格式 (受 grammar 约束)
lex: exact keyword match
vec: semantic equivalent
hyde: hypothetical document

// 示例
查询: "how to deploy"
扩展:
  lex: "deployment configuration"
  vec: "production deployment process"
  hyde: "Information about deploying applications to production"
```

#### 1.4.2 RRF 融合

Reciprocal Rank Fusion 算法：

```typescript
// 对每个结果列表
for each result in results {
  rrfScore = Σ(1 / (k + rank + 1))  // k=60
  if (rank == 1) rrfScore += 0.05     // Top-rank 奖励
  if (rank <= 3) rrfScore += 0.02
}

// 原始查询权重加倍
rrfScore_original *= 2
```

#### 1.4.3 位置感知融合

根据检索结果的 RRF 排名决定混合比例：

```typescript
// 高排名结果信任检索，低排名信任重排
const rrfWeight = rank <= 3 ? 0.75 : rank <= 10 ? 0.60 : 0.40;
const blendedScore = rrfWeight * rrfScore + (1 - rrfWeight) * rerankerScore;
```

### 1.5 上下文管理

QMD 支持**分层上下文**系统：

```yaml
# ~/.config/qmd/index.yml

global_context: "My personal knowledge base for projects"

collections:
  notes:
    path: ~/Documents/notes
    pattern: "**/*.md"
    context:
      "/": "Personal notes and ideas"
      "/work": "Work-related notes"
      "/projects": "Project documentation"

  docs:
    path: ~/work/docs
    context:
      "/api": "API reference documentation"
      "/guides": "User guides and tutorials"
```

**上下文查找逻辑：**

1. 虚拟路径解析：`qmd://collection/path/file.md`
2. 最长前缀匹配：选择最具体的上下文描述
3. 合并：全局上下文 + 集合上下文 + 路径上下文

**示例：**

```
文件: qmd://docs/api/endpoints.md
匹配上下文:
  - 全局: "My personal knowledge base"
  - 集合: "/api" → "API reference documentation"
最终: "My personal knowledge base. API reference documentation."
```

### 1.6 LLM 模型使用

| 模型 | 用途 | 大小 |
|-------|------|------|
| embeddinggemma-300M | 向量嵌入 | ~300MB |
| qwen3-reranker-0.6b | 文档重排 | ~640MB |
| qmd-query-expansion-1.7B | 查询扩展 | ~1.1GB |

**模型管理策略：**

1. **懒加载**：模型按需加载
2. **会话隔离**：每个 LLM 调用使用独立上下文
3. **活动超时**：5 分钟无活动后卸载上下文（保留模型）
4. **并行上下文**：根据 VRAM 自动创建多个嵌入上下文

---

## 二、Nanobot-Go 当前记忆实现

### 2.1 现有架构

```go
// agent/memory.go
type MemoryStore struct {
    workspace  string
    memoryDir  string
    memoryFile string
}

// 存储方式
workspace/memory/
  ├── MEMORY.md           // 长期记忆
  ├── 2026-02-21.md     // 每日笔记
  ├── 2026-02-20.md
  └── ...
```

### 2.2 当前功能

| 功能 | 实现 | QMD 对应 |
|------|--------|-----------|
| 文件存储 | ✅ 按日期分文件 | N/A |
| 长期记忆 | ✅ MEMORY.md | documents + content |
| 读取最近 N 天 | ✅ GetRecentMemories() | collections + FTS |
| 上下文构建 | ✅ BuildSystemPrompt() | path_contexts |
| 全文搜索 | ❌ 无 | documents_fts (BM25) |
| 向量搜索 | ❌ 无 | vectors_vec |
| 智能分块 | ❌ 无 | smart chunking |
| 查询扩展 | ❌ 无 | LLM expand |
| 结果重排 | ❌ 无 | cross-encoder rerank |

---

## 三、集成方案设计

### 3.1 整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│              Nanobot-Go 记忆系统 (集成后)                 │
├─────────────────────────────────────────────────────────────────┤
│                                                          │
│  文档输入 ──► 智能分块 ──► SQLite 存储器            │
│                 │                    │                         │
│                 │                    ├─ documents 表             │
│                 │                    ├─ content 表 (去重)       │
│                 │                    ├─ documents_fts (BM25)    │
│                 │                    ├─ content_vectors (分块)    │
│                 │                    └─ vectors_vec (向量)       │
│                 │                    │                         │
│                 │                    ▼                         │
│  用户查询 ──► 查询扩展 ──► 混合搜索                   │
│                   │               │                             │
│                   │               ├─ BM25 检索              │
│                   │               ├─ 向量检索                │
│                   │               └─ RRF 融合                │
│                   │               │                             │
│                   │               ▼                             │
│                   └── LLM 重排 ──► 位置感知融合             │
│                                    │                             │
│                                    ▼                             │
│                              相关性排序结果                   │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 数据模型设计

#### 3.2.1 SQLite Schema

创建 `internal/memory/schema.go`:

```go
package memory

const SchemaDDL = `
-- 集合表
CREATE TABLE IF NOT EXISTS collections (
    name TEXT PRIMARY KEY,
    path TEXT NOT NULL,
    pattern TEXT NOT NULL DEFAULT '**/*.md',
    context TEXT,                    -- JSON 字符串存储上下文映射
    update_cmd TEXT,                 -- 可选的更新命令
    include_by_default INTEGER DEFAULT 1
);

-- 文档表
CREATE TABLE IF NOT EXISTS documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    collection TEXT NOT NULL,
    path TEXT NOT NULL,
    title TEXT NOT NULL,
    hash TEXT NOT NULL,
    created_at TEXT NOT NULL,
    modified_at TEXT NOT NULL,
    active INTEGER NOT NULL DEFAULT 1,
    FOREIGN KEY (hash) REFERENCES content(hash) ON DELETE CASCADE,
    UNIQUE(collection, path)
);

CREATE INDEX IF NOT EXISTS idx_documents_collection ON documents(collection, active);
CREATE INDEX IF NOT EXISTS idx_documents_hash ON documents(hash);

-- 内容表（去重）
CREATE TABLE IF NOT EXISTS content (
    hash TEXT PRIMARY KEY,
    doc TEXT NOT NULL,
    created_at TEXT NOT NULL
);

-- 全文搜索索引 (BM25)
CREATE VIRTUAL TABLE IF NOT EXISTS documents_fts USING fts5(
    filepath, title, body,
    tokenize='porter unicode61'
);

-- 触发器：同步 documents 到 documents_fts
CREATE TRIGGER IF NOT EXISTS documents_ai AFTER INSERT ON documents
WHEN new.active = 1
BEGIN
    INSERT INTO documents_fts(rowid, filepath, title, body)
    SELECT
        new.id,
        new.collection || '/' || new.path,
        new.title,
        (SELECT doc FROM content WHERE hash = new.hash)
    WHERE new.active = 1;
END;

CREATE TRIGGER IF NOT EXISTS documents_ad AFTER DELETE ON documents BEGIN
    DELETE FROM documents_fts WHERE rowid = old.id;
END;

CREATE TRIGGER IF NOT EXISTS documents_au AFTER UPDATE ON documents BEGIN
    DELETE FROM documents_fts WHERE rowid = old.id AND new.active = 0;
    INSERT OR REPLACE INTO documents_fts(rowid, filepath, title, body)
    SELECT
        new.id,
        new.collection || '/' || new.path,
        new.title,
        (SELECT doc FROM content WHERE hash = new.hash)
    WHERE new.active = 1;
END;

-- 向量分块表
CREATE TABLE IF NOT EXISTS content_vectors (
    hash TEXT NOT NULL,
    seq INTEGER NOT NULL DEFAULT 0,
    pos INTEGER NOT NULL DEFAULT 0,
    model TEXT NOT NULL,
    embedded_at TEXT NOT NULL,
    PRIMARY KEY (hash, seq)
);

-- LLM 缓存表
CREATE TABLE IF NOT EXISTS llm_cache (
    cache_key TEXT PRIMARY KEY,
    result TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_llm_cache_created ON llm_cache(created_at);
`
```

#### 3.2.2 Go 数据结构

```go
// internal/memory/types.go

package memory

import (
    "encoding/json"
    "time"
)

// Collection 表示一个文档集合
type Collection struct {
    Name              string            `json:"name"`
    Path              string            `json:"path"`
    Pattern           string            `json:"pattern"`
    Context           map[string]string  `json:"context"`           // 路径 → 描述映射
    UpdateCmd         string            `json:"update_cmd,omitempty"`
    IncludeByDefault  bool              `json:"include_by_default"`
}

// Document 表示一个文档
type Document struct {
    ID         int64      `json:"id"`
    Collection string    `json:"collection"`
    Path       string     `json:"path"`
    Title      string     `json:"title"`
    Hash       string     `json:"hash"`
    CreatedAt  time.Time  `json:"created_at"`
    ModifiedAt time.Time  `json:"modified_at"`
    Active     bool       `json:"active"`
}

// Content 文档内容（去重）
type Content struct {
    Hash      string    `json:"hash"`
    Doc       string    `json:"doc"`
    CreatedAt time.Time `json:"created_at"`
}

// Chunk 文档分块
type Chunk struct {
    Hash      string    `json:"hash"`
    Seq       int       `json:"seq"`
    Pos       int       `json:"pos"`
    Model     string    `json:"model"`
    EmbeddedAt time.Time `json:"embedded_at"`
    // 向量数据存储在外部或扩展字段
    Vector    []float64 `json:"vector,omitempty"`
}

// BreakPoint 分块断点
type BreakPoint struct {
    Pos   int    `json:"pos"`
    Score int    `json:"score"`
    Type  string `json:"type"`
}

// SearchResult 搜索结果
type SearchResult struct {
    Filepath   string  `json:"filepath"`
    Title      string  `json:"title"`
    Score      float64 `json:"score"`
    Source     string  `json:"source"` // "fts" | "vec" | "hybrid"
    ChunkPos   *int    `json:"chunk_pos,omitempty"`
    Snippet    string  `json:"snippet"`
}

// ExpandedQuery 扩展查询
type ExpandedQuery struct {
    Type string `json:"type"` // "lex" | "vec" | "hyde"
    Text string `json:"text"`
}

// ContextEntry 上下文条目
type ContextEntry struct {
    Path    string `json:"path"`
    Context string `json:"context"`
}

// MarshalContext 将上下文映射转换为 JSON 字符串
func MarshalContext(ctx map[string]string) string {
    data, _ := json.Marshal(ctx)
    return string(data)
}

// UnmarshalContext 将 JSON 字符串转换为上下文映射
func UnmarshalContext(data string) map[string]string {
    var ctx map[string]string
    json.Unmarshal([]byte(data), &ctx)
    if ctx == nil {
        return make(map[string]string)
    }
    return ctx
}
```

### 3.3 核心模块设计

#### 3.3.1 智能分块器

创建 `internal/memory/chunk.go`:

```go
package memory

import (
    "regexp"
    "unicode"
)

// ChunkConfig 分块配置
var ChunkConfig = struct {
    SizeTokens     int  // 900 tokens
    OverlapTokens  int  // 135 tokens (15%)
    WindowTokens   int  // 200 tokens
    SizeChars      int  // 3600 chars (900 * 4)
    OverlapChars   int  // 540 chars
    WindowChars    int  // 800 chars
}{
    SizeTokens:     900,
    OverlapTokens:  135,
    WindowTokens:   200,
    SizeChars:      3600,
    OverlapChars:   540,
    WindowChars:    800,
}

// breakPatterns 断点模式（按优先级排序）
var breakPatterns = []struct {
    regex *regexp.Regexp
    score int
    name  string
}{
    {regexp.MustCompile(`\n#{1}(?!#)`), 100, "h1"},         // # but not ##
    {regexp.MustCompile(`\n#{2}(?!#)`), 90, "h2"},          // ## but not ###
    {regexp.MustCompile(`\n#{3}(?!#)`), 80, "h3"},          // ### but not ####
    {regexp.MustCompile(`\n#{4}(?!#)`), 70, "h4"},          // ####
    {regexp.MustCompile(`\n#{5}(?!#)`), 60, "h5"},          // #####
    {regexp.MustCompile(`\n#{6}(?!#)`), 50, "h6"},          // ######
    {regexp.MustCompile(`\n```), 80, "codeblock"},              // code block boundary
    {regexp.MustCompile(`\n(?:---|\*\*\*|___)\s*\n`), 60, "hr"}, // horizontal rule
    {regexp.MustCompile(`\n\n+`), 20, "blank"},                // paragraph boundary
    {regexp.MustCompile(`\n[-*]\s`), 5, "list"},                 // unordered list
    {regexp.MustCompile(`\n\d+\.\s`), 5, "numlist"},             // ordered list
    {regexp.MustCompile(`\n`), 1, "newline"},                   // minimal break
}

// CodeFenceRegion 代码块区域
type CodeFenceRegion struct {
    Start int
    End   int
}

// ScanBreakPoints 扫描文档中的所有断点
func ScanBreakPoints(text string) []BreakPoint {
    seen := make(map[int]BreakPoint)

    for _, bp := range breakPatterns {
        matches := bp.regex.FindAllStringIndex(text, -1)
        for _, m := range matches {
            pos := m[0]
            existing, ok := seen[pos]
            if !ok || bp.score > existing.Score {
                seen[pos] = BreakPoint{
                    Pos:   pos,
                    Score:  bp.score,
                    Type:   bp.name,
                }
            }
        }
    }

    // 转换为有序切片
    var points []BreakPoint
    for _, bp := range seen {
        points = append(points, bp)
    }
    // 按位置排序
    sort.Slice(points, func(i, j int) bool {
        return points[i].Pos < points[j].Pos
    })

    return points
}

// FindCodeFences 查找代码块区域
func FindCodeFences(text string) []CodeFenceRegion {
    var regions []CodeFenceRegion
    fencePattern := regexp.MustCompile(`\n````)
    matches := fencePattern.FindAllStringIndex(text, -1)

    inFence := false
    var fenceStart int

    for _, m := range matches {
        if !inFence {
            fenceStart = m[0]
            inFence = true
        } else {
            regions = append(regions, CodeFenceRegion{
                Start: fenceStart,
                End:   m[1],
            })
            inFence = false
        }
    }

    // 处理未闭合的代码块
    if inFence {
        regions = append(regions, CodeFenceRegion{
            Start: fenceStart,
            End:   len(text),
        })
    }

    return regions
}

// IsInsideCodeFence 检查位置是否在代码块内
func IsInsideCodeFence(pos int, fences []CodeFenceRegion) bool {
    for _, f := range fences {
        if pos > f.Start && pos < f.End {
            return true
        }
    }
    return false
}

// FindBestCutoff 查找最佳截断位置
func FindBestCutoff(
    breakPoints []BreakPoint,
    targetPos int,
    windowChars int,
    codeFences []CodeFenceRegion,
) int {
    windowStart := targetPos - windowChars
    bestScore := -1.0
    bestPos := targetPos

    for _, bp := range breakPoints {
        if bp.Pos < windowStart {
            continue
        }
        if bp.Pos > targetPos {
            break
        }

        // 跳过代码块内的断点
        if IsInsideCodeFence(bp.Pos, codeFences) {
            continue
        }

        // 距离衰减（平方衰减）
        distance := targetPos - bp.Pos
        normalizedDist := float64(distance) / float64(windowChars)
        decayFactor := 0.7
        multiplier := 1.0 - (normalizedDist * normalizedDist * decayFactor)
        finalScore := float64(bp.Score) * multiplier

        if finalScore > bestScore {
            bestScore = finalScore
            bestPos = bp.Pos
        }
    }

    return bestPos
}

// ChunkDocumentByChars 按字符分块文档
func ChunkDocumentByChars(text string) ([]Chunk, error) {
    breakPoints := ScanBreakPoints(text)
    codeFences := FindCodeFences(text)

    var chunks []Chunk
    pos := 0
    seq := 0

    for pos < len(text) {
        targetPos := pos + ChunkConfig.SizeChars

        if targetPos >= len(text) {
            // 最后一部分
            chunks = append(chunks, Chunk{
                Seq: seq,
                Pos: pos,
                Text: text[pos:],
            })
            break
        }

        // 查找最佳断点
        cutoff := FindBestCutoff(breakPoints, targetPos, ChunkConfig.WindowChars, codeFences)

        chunks = append(chunks, Chunk{
            Seq: seq,
            Pos: pos,
            Text: text[pos:cutoff],
        })

        pos = cutoff - ChunkConfig.OverlapChars
        seq++
    }

    return chunks, nil
}

// EstimateTokens 估算 token 数量（粗略：4 字符 = 1 token）
func EstimateTokens(text string) int {
    // 更准确的估算可以结合中文字符
    runes := []rune(text)
    count := 0
    for _, r := range runes {
        if unicode.Is(unicode.Han, r) {
            count++ // 中文通常 1 字符 ≈ 1 token
        } else {
            count++ // 其他字符 4 字符 ≈ 1 token（下面除以 4）
        }
    }
    return (count + len(text)/4) / 2
}
```

#### 3.3.2 存储接口

创建 `internal/memory/store.go`:

```go
package memory

import (
    "database/sql"
    "embed"
    "time"

    _ "github.com/mattn/go-sqlite3"
)

// Store 记忆存储接口
type Store struct {
    db        *sql.DB
    dbPath    string
    embedding EmbeddingService
}

// EmbeddingService 嵌入服务接口
type EmbeddingService interface {
    Embed(text string) ([]float64, error)
    EmbedBatch(texts []string) ([][]float64, error)
    Dimensions() int
}

// NewStore 创建存储实例
func NewStore(dbPath string, embedding EmbeddingService) (*Store, error) {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, err
    }

    // 执行 schema
    if _, err := db.Exec(SchemaDDL); err != nil {
        return nil, err
    }

    return &Store{
        db:        db,
        dbPath:    dbPath,
        embedding: embedding,
    }, nil
}

// Close 关闭存储
func (s *Store) Close() error {
    return s.db.Close()
}

// AddCollection 添加集合
func (s *Store) AddCollection(col Collection) error {
    ctx := MarshalContext(col.Context)
    _, err := s.db.Exec(`
        INSERT OR REPLACE INTO collections (name, path, pattern, context, update_cmd, include_by_default)
        VALUES (?, ?, ?, ?, ?, ?)
    `, col.Name, col.Path, col.Pattern, ctx, col.UpdateCmd, col.IncludeByDefault)
    return err
}

// GetCollections 获取所有集合
func (s *Store) GetCollections() ([]Collection, error) {
    rows, err := s.db.Query(`SELECT name, path, pattern, context, update_cmd, include_by_default FROM collections`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var cols []Collection
    for rows.Next() {
        var col Collection
        var ctxStr string
        var updateCmd, includeDefault sql.NullString, sql.NullInt64

        err := rows.Scan(&col.Name, &col.Path, &col.Pattern, &ctxStr, &updateCmd, &includeDefault)
        if err != nil {
            return nil, err
        }

        col.Context = UnmarshalContext(ctxStr)
        col.UpdateCmd = updateCmd.String
        col.IncludeByDefault = includeDefault.Valid && includeDefault.Int64 != 0

        cols = append(cols, col)
    }

    return cols, nil
}

// UpsertDocument 插入或更新文档
func (s *Store) UpsertDocument(col *Document) error {
    hash := col.Hash
    now := time.Now().Format(time.RFC3339)

    // 先插入/更新 content（去重）
    if _, err := s.db.Exec(`
        INSERT OR REPLACE INTO content (hash, doc, created_at)
        VALUES (?, ?, ?)
    `, hash, col.Doc, now); err != nil {
        return err
    }

    // 插入或更新 documents
    active := 1
    if !col.Active {
        active = 0
    }
    _, err := s.db.Exec(`
        INSERT OR REPLACE INTO documents (collection, path, title, hash, created_at, modified_at, active)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `, col.Collection, col.Path, col.Title, hash, col.CreatedAt.Format(time.RFC3339),
        col.ModifiedAt.Format(time.RFC3339), active)
    return err
}

// SearchBM25 BM25 全文搜索
func (s *Store) SearchBM25(query string, collectionName string, limit int) ([]SearchResult, error) {
    var sql string
    var args []any

    if collectionName != "" {
        sql = `
            SELECT d.path, d.title, bm25(documents_fts, 10.0, 1.0) as score, 'fts' as source
            FROM documents_fts f
            JOIN documents d ON f.rowid = d.id
            WHERE f MATCH ? AND d.collection = ? AND d.active = 1
            ORDER BY score ASC
            LIMIT ?
        `
        args = []any{query, collectionName, limit}
    } else {
        sql = `
            SELECT d.path, d.title, bm25(documents_fts, 10.0, 1.0) as score, 'fts' as source
            FROM documents_fts f
            JOIN documents d ON f.rowid = d.id
            WHERE f MATCH ? AND d.active = 1
            ORDER BY score ASC
            LIMIT ?
        `
        args = []any{query, limit}
    }

    rows, err := s.db.Query(sql, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var results []SearchResult
    for rows.Next() {
        var r SearchResult
        var bm25Score float64
        if err := rows.Scan(&r.Filepath, &r.Title, &bm25Score, &r.Source); err != nil {
            return nil, err
        }
        // 转换 BM25 分数为 0-1 范围
        r.Score = math.Abs(bm25Score) / (1 + math.Abs(bm25Score))
        results = append(results, r)
    }

    return results, nil
}

// GetDocumentsForEmbedding 获取需要嵌入的文档
func (s *Store) GetDocumentsForEmbedding(collectionName string) ([]Document, error) {
    var sql string
    var args []any

    if collectionName != "" {
        sql = `
            SELECT d.id, d.collection, d.path, d.title, d.hash, c.doc, d.active
            FROM documents d
            JOIN content c ON d.hash = c.hash
            WHERE d.collection = ? AND d.active = 1
            AND NOT EXISTS (
                SELECT 1 FROM content_vectors v
                WHERE v.hash = d.hash AND v.seq = 0
            )
        `
        args = []any{collectionName}
    } else {
        sql = `
            SELECT d.id, d.collection, d.path, d.title, d.hash, c.doc, d.active
            FROM documents d
            JOIN content c ON d.hash = c.hash
            WHERE d.active = 1
            AND NOT EXISTS (
                SELECT 1 FROM content_vectors v
                WHERE v.hash = d.hash AND v.seq = 0
            )
        `
    }

    rows, err := s.db.Query(sql, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var docs []Document
    for rows.Next() {
        var d Document
        var docStr string
        var active int
        if err := rows.Scan(&d.ID, &d.Collection, &d.Path, &d.Title, &d.Hash, &docStr, &active); err != nil {
            return nil, err
        }
        d.Doc = docStr
        d.Active = active != 0
        docs = append(docs, d)
    }

    return docs, nil
}
```

#### 3.3.3 向量嵌入集成

创建 `internal/memory/embedding.go`:

```go
package memory

import (
    "context"
    "fmt"
)

// LocalEmbeddingService 本地嵌入服务（使用 llama.cpp 或类似）
type LocalEmbeddingService struct {
    modelPath string
    dimensions int
}

func NewLocalEmbeddingService(modelPath string, dimensions int) *LocalEmbeddingService {
    return &LocalEmbeddingService{
        modelPath: modelPath,
        dimensions: dimensions,
    }
}

func (s *LocalEmbeddingService) Embed(text string) ([]float64, error) {
    // 使用 llama.cpp 或 node-llama-cpp 的 Go 绑定
    // 示例使用外部进程调用
    return []float64{}, nil
}

func (s *LocalEmbeddingService) EmbedBatch(texts []string) ([][]float64, error) {
    results := make([][]float64, len(texts))
    for i, text := range texts {
        vec, err := s.Embed(text)
        if err != nil {
            return nil, err
        }
        results[i] = vec
    }
    return results, nil
}

func (s *LocalEmbeddingService) Dimensions() int {
    return s.dimensions
}

// OpenAIEmbeddingService OpenAI 嵌入服务（备选）
type OpenAIEmbeddingService struct {
    apiKey string
    model  string
    client *http.Client
}

func NewOpenAIEmbeddingService(apiKey, model string) *OpenAIEmbeddingService {
    return &OpenAIEmbeddingService{
        apiKey: apiKey,
        model:  model,
        client: &http.Client{},
    }
}

func (s *OpenAIEmbeddingService) Embed(ctx context.Context, text string) ([]float64, error) {
    // 调用 OpenAI Embeddings API
    return []float64{}, nil
}

func (s *OpenAIEmbeddingService) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
    // 批量调用 API
    return [][]float64{}, nil
}

func (s *OpenAIEmbeddingService) Dimensions() int {
    return 1536 // text-embedding-ada-002
}
```

### 3.4 搜索流程实现

创建 `internal/memory/search.go`:

```go
package memory

import (
    "math"
    "sort"
)

// RRFK RRF 融合常数
const RRFK = 60

// SearchConfig 搜索配置
type SearchConfig struct {
    MinScore          float64
    MaxCandidates     int
    EnableVector      bool
    EnableExpansion   bool
    EnableReranking  bool
}

// DefaultSearchConfig 默认搜索配置
var DefaultSearchConfig = SearchConfig{
    MinScore:         0.2,
    MaxCandidates:    30,
    EnableVector:     true,
    EnableExpansion:   true,
    EnableReranking:  true,
}

// HybridSearch 混合搜索
func (s *Store) HybridSearch(
    ctx context.Context,
    query string,
    config SearchConfig,
) ([]SearchResult, error) {
    var allResults map[string]struct {
        result SearchResult
        rrfScores map[string]float64
        topRank   map[string]int
    }
    allResults = make(map[string]struct {
        result SearchResult
        rrfScores map[string]float64
        topRank   map[string]int
    })

    queryCount := 0

    // 原始查询
    if config.EnableVector {
        ftsResults, _ := s.SearchBM25(query, "", config.MaxCandidates)
        for i, r := range ftsResults {
            s.addToRRF(allResults, r, queryCount, i*2) // ×2 权重
        }
        queryCount++

        vecResults, _ := s.SearchVector(ctx, query, "", config.MaxCandidates)
        for i, r := range vecResults {
            s.addToRRF(allResults, r, queryCount, i)
        }
        queryCount++
    }

    // 查询扩展（可选）
    if config.EnableExpansion {
        expanded, err := s.ExpandQuery(ctx, query)
        if err == nil {
            for _, eq := range expanded {
                if eq.Type == "vec" || eq.Type == "hyde" {
                    vecResults, _ := s.SearchVector(ctx, eq.Text, "", config.MaxCandidates)
                    for i, r := range vecResults {
                        s.addToRRF(allResults, r, queryCount, i)
                    }
                    queryCount++
                }
                if eq.Type == "lex" {
                    ftsResults, _ := s.SearchBM25(eq.Text, "", config.MaxCandidates)
                    for i, r := range ftsResults {
                        s.addToRRF(allResults, r, queryCount, i)
                    }
                    queryCount++
                }
            }
        }
    }

    // 融合并排序
    var finalResults []SearchResult
    for key, data := range allResults {
        // 计算 RRF 分数
        rrfScore := 0.0
        for _, score := range data.rrfScores {
            rrfScore += score
        }

        // Top-rank 奖励
        for _, rank := range data.topRank {
            if rank == 1 {
                rrfScore += 0.05
            } else if rank <= 3 {
                rrfScore += 0.02
            }
        }

        data.result.Score = rrfScore

        // 位置感知融合（如果有重排分数）
        if data.result.RerankerScore > 0 {
            rrfWeight := 0.40 // 默认使用较低权重
            if data.result.TopRRFRank <= 3 {
                rrfWeight = 0.75
            } else if data.result.TopRRFRank <= 10 {
                rrfWeight = 0.60
            }
            data.result.Score = rrfWeight*rrfScore + (1-rrfWeight)*data.result.RerankerScore
        }

        finalResults = append(finalResults, data.result)
    }

    // 排序并过滤
    sort.Slice(finalResults, func(i, j int) bool {
        return finalResults[i].Score > finalResults[j].Score
    })

    var filtered []SearchResult
    for _, r := range finalResults {
        if r.Score >= config.MinScore {
            filtered = append(filtered, r)
        }
    }

    if len(filtered) > config.MaxCandidates {
        filtered = filtered[:config.MaxCandidates]
    }

    return filtered, nil
}

// addToRRF 添加到 RRF 融合
func (s *Store) addToRRF(
    results map[string]struct {
        result SearchResult
        rrfScores map[string]float64
        topRank   map[string]int
    },
    r SearchResult,
    queryIndex, rank int,
) {
    key := r.Filepath
    data, exists := results[key]
    if !exists {
        data = struct {
            result SearchResult
            rrfScores map[string]float64
            topRank   map[string]int
        }{
            rrfScores: make(map[string]float64),
            topRank:   make(map[string]int),
        }
        data.result = r
    }

    // RRF 分数计算
    rrfScore := 1.0 / float64(RRFK+rank+1)
    data.rrfScores[fmt.Sprintf("q%d", queryIndex)] = rrfScore

    // 记录 top rank
    if _, ok := data.topRank[fmt.Sprintf("q%d", queryIndex)]; !ok || rank < data.topRank[fmt.Sprintf("q%d", queryIndex)] {
        data.topRank[fmt.Sprintf("q%d", queryIndex)] = rank
    }

    results[key] = data
}
```

### 3.5 上下文管理增强

扩展现有 `agent/context.go`：

```go
// 在 ContextBuilder 中添加
type ContextBuilder struct {
    // ... 现有字段
    memoryStore *memory.Store  // 新增
}

// GetRelevantMemoryContext 获取与查询相关的记忆上下文
func (c *ContextBuilder) GetRelevantMemoryContext(ctx context.Context, query string, limit int) (string, error) {
    if c.memoryStore == nil {
        return "", nil
    }

    // 使用混合搜索获取相关记忆
    results, err := c.memoryStore.HybridSearch(ctx, query, memory.SearchConfig{
        MaxCandidates:    limit,
        MinScore:         0.3,
        EnableVector:     true,
        EnableExpansion:   false, // 简化：不扩展查询
        EnableReranking:  false, // 简化：不重排
    })

    if err != nil || len(results) == 0 {
        return "", nil
    }

    // 构建上下文字符串
    var parts []string
    parts = append(parts, fmt.Sprintf("## 相关记忆 (%d 条)", len(results)))

    for _, r := range results {
        parts = append(parts, fmt.Sprintf(
            "- **%s** (%.2f%%): %s",
            r.Title, r.Score*100, r.Snippet,
        ))
    }

    return strings.Join(parts, "\n"), nil
}
```

---

## 四、集成路线图

### 阶段 1: 基础设施 (Week 1)

- [ ] 创建 `internal/memory` 包
- [ ] 实现 SQLite schema 和存储层
- [ ] 添加 go-sqlite3 依赖
- [ ] 编写单元测试

### 阶段 2: 文档索引 (Week 2)

- [ ] 实现智能分块算法
- [ ] 集成嵌入服务（先使用外部 API）
- [ ] 实现 BM25 FTS 索引
- [ ] 实现文档导入流程

### 阶段 3: 基础搜索 (Week 3)

- [ ] 实现 BM25 全文搜索
- [ ] 实现向量相似度搜索
- [ ] 实现 RRF 融合
- [ ] 集成到 ContextBuilder

### 阶段 4: 高级功能 (Week 4)

- [ ] 实现查询扩展
- [ ] 实现 LLM 重排
- [ ] 添加上下文管理系统
- [ ] 性能优化和缓存

### 阶段 5: 本地 LLM 集成 (Week 5-6)

- [ ] 评估本地嵌入模型（llama.cpp）
- [ ] 集成本地重排模型
- [ ] 实现 go-llama-cpp 绑定或使用 CGO
- [ ] 端到端测试

---

## 五、技术决策点

### 5.1 嵌入服务选择

| 方案 | 优点 | 缺点 | 推荐度 |
|------|------|------|--------|
| OpenAI API | 简单集成，高质量 | 成本高，依赖网络 | ★★★☆ |
| 本地 llama.cpp | 免费用，隐私保护 | 需要编译，模型大 | ★★★★ |
| Ollama API | 易集成，本地运行 | 额外服务 | ★★★☆ |
| Qdrant/Weaviate | 专用向量数据库 | 复杂度高 | ★★☆☆ |

**推荐：** 先用 OpenAI API 快速验证，再迁移到本地 llama.cpp

### 5.2 向量数据库选择

| 方案 | 优点 | 缺点 |
|------|------|------|
| sqlite-vec | 单文件，简单，与 FTS 同库 | 需要编译扩展 |
| Qdrant | 独立服务，功能丰富 | 部署复杂 |
| Weaviate | 云原生 | 资源消耗大 |
| PostgreSQL pgvector | 标准数据库，稳定 | 需要额外服务 |

**推荐：** sqlite-vec（与 QMD 一致）

### 5.3 中文支持

QMD 原生设计针对英文。针对中文需要调整：

1. **分块配置**：中文字符 token 比例不同
   ```go
   // 中文：1 字符 ≈ 1 token，英文：4 字符 ≈ 1 token
   func EstimateTokens(text string) int {
       runes := []rune(text)
       count := 0
       for _, r := range runes {
           if unicode.Is(unicode.Han, r) {
               count++
           } else {
               count++ // 需要更精确的 tokenizer
           }
       }
       // 混合估算
       return (count + len(text)/4) / 2
   }
   ```

2. **分词器**：考虑使用 `jieba` 或 `gosegment` 进行中文分词

3. **嵌入模型**：使用支持中文的模型（如 `bge-large-zh`）

---

## 六、依赖添加

```bash
# go.mod 添加
go get github.com/mattn/go-sqlite3
go get github.com/ncruces/go-sqlite3
go get github.com/twinj/ego
go get github.com/yanyiwu/gojieba  # 中文分词（可选）

# 如果使用本地 llama.cpp
go get github.com/ggerganov/llama.cpp
```

---

## 七、测试策略

```go
// internal/memory/search_test.go

func TestRRFFusion(t *testing.T) {
    results := []SearchResult{...}
    fused := fuseRRF(results, RRFK)
    // 验证融合结果
}

func TestSmartChunking(t *testing.T) {
    markdown := "# Title\n\nContent...\n## Section\n\nMore content"
    chunks := ChunkDocumentByChars(markdown)
    // 验证分块边界在标题处
}

func TestHybridSearch(t *testing.T) {
    // 测试 BM25 + Vector + RRF 完整流程
}

func BenchmarkEmbedding(b *testing.B) {
    // 性能基准测试
}
```

---

## 八、参考资源

- [QMD GitHub](https://github.com/tobi/qmd)
- [sqlite-vec](https://github.com/asg017/sqlite-vec)
- [BM25 算法说明](https://en.wikipedia.org/wiki/Okapi_BM25)
- [RRF 论文](https://plg.uwaterloo.ca/~gvcormac/cormacksig2008-38up.pdf)
- [Llama.cpp Go 绑定](https://github.com/go-skynet/llama.cpp)

---

## 九、总结

QMD 的记忆处理核心思路可以总结为：

1. **智能分块**：基于自然边界的文档分块，保持语义完整性
2. **混合索引**：BM25 全文搜索 + 向量语义搜索
3. **查询扩展**：使用 LLM 生成查询变体，提高召回率
4. **RRF 融合**：多路检索结果融合，平衡不同信号
5. **LLM 重排**：使用 cross-encoder 精细排序
6. **位置感知**：根据检索排名动态调整融合比例
7. **上下文分层**：全局 + 集合 + 路径三级上下文
8. **单库存储**：所有数据存储在单个 SQLite 数据库

集成到 nanobot-go 需要：
1. 建立 SQLite 数据库层
2. 实现智能分块算法
3. 集成嵌入服务
4. 实现混合搜索流程
5. 增强 ContextBuilder 支持相关记忆检索
6. 适配中文支持

建议按照上述路线图分阶段实施，每个阶段完成后验证功能再继续下一阶段。
