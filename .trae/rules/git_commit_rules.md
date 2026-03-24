# Git 提交规范

本文档定义了项目的 Git 提交信息规范，采用 [Conventional Commits](https://www.conventionalcommits.org/) 标准。

## 1. 提交信息格式

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### 基本规则

- 提交信息使用**中文**撰写
- 类型(type)小写
- 标题行不超过 72 字符
- 描述首字母小写（除非是专有名词或缩写）
- 使用祈使语气（如 "add" 而不是 "added"）

## 2. Type 类型

| Type       | 描述                       |
| ---------- | ------------------------ |
| `feat`     | 新功能                      |
| `fix`      | 修复 bug                   |
| `docs`     | 仅文档变更                    |
| `style`    | 不影响代码含义的格式修改（空格、格式化、分号等） |
| `refactor` | 代码重构（非功能修改）              |
| `perf`     | 性能优化                     |
| `test`     | 测试相关                     |
| `build`    | 构建系统或依赖变更                |
| `ci`       | CI/CD 配置变更               |
| `chore`    | 其他不修改 src/test 的变更       |

## 3. Scope 可选范围

使用括号包裹，指定影响范围：

```
feat(api): 添加用户认证功能
fix(ui): 修复按钮样式问题
docs(readme): 更新安装说明
```

### 常用 Scope

- `api` - API 接口
- `ui` - 用户界面
- `db` - 数据库相关
- `auth` - 认证授权
- `config` - 配置
- `deps` - 依赖
- `core` - 核心业务逻辑

## 4. 示例

### 功能开发

```
feat(auth): 添加 JWT 令牌刷新机制
```

### Bug 修复

```
fix(api): 修复用户登录失败时返回错误码不正确的问题
```

### 重构

```
refactor(service): 简化订单处理逻辑
```

### 文档更新

```
docs(readme): 添加项目架构说明
```

### 多行提交

```
fix(auth): 修复令牌过期后无法自动刷新

- 添加 token refresh 逻辑
- 处理 refresh 失败的降级方案
- 添加相关单元测试

Closes #123
```

## 5. 提交前检查

- [ ] 提交信息清晰描述了变更内容
- [ ] 遵循格式：`type(scope): description`
- [ ] 使用祈使语气
- [ ] 标题不超过 72 字符
- [ ] 关联的 Issue 在 Footer 中引用

## 6. Footer 规范

### 关联 Issue

```
Closes #123
Closes #123, #456
Fixes #123
```

### 破坏性变更

```
BREAKING CHANGE: 移除了旧的认证接口
```

## 7. 提交粒度原则

**一次提交，一个关注点**

- ✅ `feat: 添加用户注册功能`
- ❌ `feat: 添加用户注册功能, 修复样式问题, 更新文档`

每个逻辑变更单独提交，便于追溯和管理。

## 8. Git Hooks（可选）

项目可配置 pre-commit hook 验证提交信息格式：

```bash
npx commitlint --edit
```

***

*最后更新: 2026-03-24*
