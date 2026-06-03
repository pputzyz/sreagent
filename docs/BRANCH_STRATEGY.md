# SREAgent 分支策略

## 分支结构

```
main (生产分支)
  ├── 只接收经过测试验证的代码
  ├── 打 tag 触发 GitHub Actions CI
  ├── 部署到生产环境
  └── 保护规则：禁止直接 push

test (测试分支)
  ├── 日常开发和测试
  ├── 自动化 QA 测试运行
  ├── Bug 修复和功能开发
  └── 通过 PR 合并到 main
```

## 工作流程

### 1. 开发阶段（test 分支）
```bash
# 切换到 test 分支
git checkout test

# 进行开发和测试
# ... 修改代码 ...

# 提交到 test
git add -A
git commit -m "feat: 新功能描述"
git push origin test
```

### 2. 测试阶段
```bash
# 在 test 分支运行自动化测试
cd tests/qa
npm test

# 运行后端测试
go test ./...

# 运行前端类型检查
cd web && npx vue-tsc --noEmit
```

### 3. 合并到 main
```bash
# 切换到 main
git checkout main

# 合并 test 分支（排除测试产物）
git merge test --no-ff -m "merge: test → main"

# 打 tag 触发 CI
git tag v4.63.0
git push origin main --tags
```

### 4. 测试产物管理
测试产物（tests/qa/test-results/、tests/qa/reports/）：
- 只存在于 test 分支
- 通过 .gitignore 排除
- 不会合并到 main
- 不会触发 CI

## 分支保护规则（GitHub Settings）

### main 分支
- [x] Require pull request reviews
- [x] Require status checks to pass
- [x] Require branches to be up to date
- [x] Include administrators

### test 分支
- [ ] 无特殊限制
- 允许直接 push
- 允许 force push

## CI/CD 触发条件

### GitHub Actions 触发条件：
- **main 分支**: 推送 `v*` tag → 构建 Docker 镜像 → 推送到 Docker Hub
- **test 分支**: 推送代码 → 运行测试（不构建镜像）
- **PR**: 运行测试 + 类型检查

### 本地测试：
- test 分支：运行完整测试套件
- main 分支：只验证构建

## 命名规范

### 分支命名：
- `main` - 生产分支
- `test` - 测试分支
- `feature/xxx` - 功能分支（从 test 分出）
- `fix/xxx` - 修复分支（从 test 分出）
- `hotfix/xxx` - 紧急修复（从 main 分出）

### Tag 命名：
- `v4.62.0` - 正式版本
- `v4.62.0-rc.1` - 候选版本
- `v4.62.0-beta.1` - 测试版本

## 紧急修复流程

如果生产环境发现紧急 Bug：
```bash
# 从 main 创建 hotfix 分支
git checkout main
git checkout -b hotfix/critical-bug

# 修复 Bug
# ... 修改代码 ...

# 合并到 main 和 test
git checkout main
git merge hotfix/critical-bug
git tag v4.62.1
git push origin main --tags

git checkout test
git merge hotfix/critical-bug
git push origin test

# 删除 hotfix 分支
git branch -d hotfix/critical-bug
```
