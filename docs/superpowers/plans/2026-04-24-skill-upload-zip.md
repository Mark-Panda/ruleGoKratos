# Skill Upload Zip Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 skill 管理上传改为仅接受 zip 压缩包，并在服务端自动按压缩包内目录结构解压到 `skills/` 根目录，而不是把 zip 文件本身落盘。

**Architecture:** 复用现有 `AdminService.UploadSkill` 接口，保持前端调用入口不变，但将后端语义调整为“上传 zip 包并解压”。服务端负责 zip 扩展名校验、base64 解码、zip 条目安全校验与解压；前端只负责限制上传文件类型和传递 zip 文件名。

**Tech Stack:** Go `archive/zip`、Kratos service、前端 TypeScript API 封装

---

### Task 1: 为 zip-only 上传补失败测试

**Files:**
- Create: `internal/service/admin_upload_skill_test.go`
- Modify: `internal/service/admin.go`
- Test: `internal/service/admin_upload_skill_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestAdminServiceUploadSkillRejectsNonZip(t *testing.T) {
	svc := &AdminService{skillRoot: t.TempDir()}
	contentBase64 := base64.StdEncoding.EncodeToString([]byte("plain text"))

	_, err := svc.UploadSkill(context.Background(), &v1.UploadSkillRequest{
		Path:          "demo.md",
		ContentBase64: contentBase64,
	})
	if err == nil || !strings.Contains(err.Error(), ".zip") {
		t.Fatalf("expected zip validation error, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run TestAdminServiceUploadSkillRejectsNonZip -count=1`
Expected: FAIL，因为当前实现会把 `demo.md` 直接写入磁盘，不会拒绝非 zip。

- [ ] **Step 3: Write minimal implementation**

```go
if strings.ToLower(filepath.Ext(safeName)) != ".zip" {
	return nil, errors.New("仅支持上传.zip压缩包")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/service -run TestAdminServiceUploadSkillRejectsNonZip -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/admin_upload_skill_test.go internal/service/admin.go
git commit -m "fix: restrict skill uploads to zip archives"
```

### Task 2: 为自动解压行为补失败测试

**Files:**
- Modify: `internal/service/admin_upload_skill_test.go`
- Modify: `internal/service/admin.go`
- Test: `internal/service/admin_upload_skill_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestAdminServiceUploadSkillExtractsZipIntoSkillRoot(t *testing.T) {
	svc := &AdminService{skillRoot: t.TempDir()}
	contentBase64 := base64.StdEncoding.EncodeToString(buildZipArchive(t, map[string]string{
		"demo/SKILL.md": "name: demo",
		"demo/config.json": `{"ok":true}`,
	}))

	reply, err := svc.UploadSkill(context.Background(), &v1.UploadSkillRequest{
		Path:          "demo.zip",
		ContentBase64: contentBase64,
	})
	if err != nil {
		t.Fatalf("UploadSkill failed: %v", err)
	}
	if got := reply.GetPath(); got != "demo" {
		t.Fatalf("expected package path demo, got %q", got)
	}
	assertFileContent(t, filepath.Join(svc.skillRoot, "demo", "SKILL.md"), "name: demo")
	assertFileContent(t, filepath.Join(svc.skillRoot, "demo", "config.json"), `{"ok":true}`)
	if _, statErr := os.Stat(filepath.Join(svc.skillRoot, "demo.zip")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected zip file not persisted, stat err=%v", statErr)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run TestAdminServiceUploadSkillExtractsZipIntoSkillRoot -count=1`
Expected: FAIL，因为当前实现只会写出 `demo.zip` 文件，不会生成 `demo/SKILL.md`。

- [ ] **Step 3: Write minimal implementation**

```go
content, err := base64.StdEncoding.DecodeString(raw)
if err != nil {
	return nil, errors.New("contentBase64格式错误")
}
packagePath, err := unzipSkillArchive(s.skillRoot, safeName, content)
if err != nil {
	return nil, err
}
return &v1.UploadSkillReply{Path: filepath.ToSlash(packagePath)}, nil
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/service -run TestAdminServiceUploadSkillExtractsZipIntoSkillRoot -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/admin_upload_skill_test.go internal/service/admin.go
git commit -m "fix: extract uploaded skill archives"
```

### Task 3: 为 zip slip 安全校验补失败测试

**Files:**
- Modify: `internal/service/admin_upload_skill_test.go`
- Modify: `internal/service/admin.go`
- Test: `internal/service/admin_upload_skill_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestAdminServiceUploadSkillRejectsUnsafeZipEntry(t *testing.T) {
	svc := &AdminService{skillRoot: t.TempDir()}
	contentBase64 := base64.StdEncoding.EncodeToString(buildZipArchive(t, map[string]string{
		"../escape.txt": "bad",
	}))

	_, err := svc.UploadSkill(context.Background(), &v1.UploadSkillRequest{
		Path:          "escape.zip",
		ContentBase64: contentBase64,
	})
	if err == nil || !strings.Contains(err.Error(), "压缩包") {
		t.Fatalf("expected unsafe zip entry error, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run TestAdminServiceUploadSkillRejectsUnsafeZipEntry -count=1`
Expected: FAIL，因为当前实现不会解析 zip，也不会拦截危险条目。

- [ ] **Step 3: Write minimal implementation**

```go
entryPath, err := sanitizeSkillArchiveEntry(file.Name)
if err != nil {
	return "", fmt.Errorf("压缩包内容不合法: %w", err)
}
targetPath := filepath.Join(skillRoot, entryPath)
if !strings.HasPrefix(targetPath, skillRootWithSep) && targetPath != skillRoot {
	return "", errors.New("压缩包内容不合法")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/service -run TestAdminServiceUploadSkillRejectsUnsafeZipEntry -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/admin_upload_skill_test.go internal/service/admin.go
git commit -m "fix: validate uploaded skill archive entries"
```

### Task 4: 前端上传入口限制 zip

**Files:**
- Modify: `flowgram/src/services/api-agent.ts`
- Test: 手动验证管理端上传交互

- [ ] **Step 1: Update upload call contract**

```ts
export const uploadSkill = async (file: File, path?: string) => {
  const uploadPath = path || file.name;
  if (!/\.zip$/i.test(uploadPath)) {
    throw new Error('仅支持上传 .zip 技能包');
  }
  // ... existing arrayBuffer -> base64 logic
};
```

- [ ] **Step 2: Verify API call shape stays compatible**

Run: 管理端选择一个 `.zip` 文件执行上传
Expected: 请求仍然发送 `{ path, contentBase64 }`，其中 `path` 为 zip 文件名

- [ ] **Step 3: Commit**

```bash
git add flowgram/src/services/api-agent.ts
git commit -m "fix: enforce zip skill uploads in admin client"
```

### Task 5: 全量验证

**Files:**
- Modify: `internal/service/admin.go`
- Modify: `internal/service/admin_upload_skill_test.go`
- Modify: `flowgram/src/services/api-agent.ts`

- [ ] **Step 1: Run focused Go tests**

```bash
go test ./internal/service -run TestAdminServiceUploadSkill -count=1
```

- [ ] **Step 2: Run package-level Go tests**

```bash
go test ./internal/service/... -count=1
```

- [ ] **Step 3: Run frontend lint/type check if available**

```bash
pnpm --dir flowgram exec tsc --noEmit
```

- [ ] **Step 4: Inspect changed behavior manually**

Run: 上传一个包含 `demo/SKILL.md` 的 zip
Expected: `skills/demo/SKILL.md` 被创建，`skills/demo.zip` 不存在
