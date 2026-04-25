package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	v1 "ruleGoKratos/api/rulego/v1"
)

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

func TestAdminServiceUploadSkillWrapsArchiveEntriesUnderArchiveName(t *testing.T) {
	svc := &AdminService{skillRoot: t.TempDir()}
	contentBase64 := base64.StdEncoding.EncodeToString(buildZipArchive(t, map[string]string{
		"SKILL.md":    "name: demo",
		"config.json": `{"ok":true}`,
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
	if _, statErr := os.Stat(filepath.Join(svc.skillRoot, "SKILL.md")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected root file not persisted, stat err=%v", statErr)
	}
}

func TestAdminServiceUploadSkillAlwaysWrapsEvenWhenZipAlreadyHasTopLevelDir(t *testing.T) {
	svc := &AdminService{skillRoot: t.TempDir()}
	contentBase64 := base64.StdEncoding.EncodeToString(buildZipArchive(t, map[string]string{
		"demo/SKILL.md":    "name: demo",
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
	assertFileContent(t, filepath.Join(svc.skillRoot, "demo", "demo", "SKILL.md"), "name: demo")
	assertFileContent(t, filepath.Join(svc.skillRoot, "demo", "demo", "config.json"), `{"ok":true}`)
	if _, statErr := os.Stat(filepath.Join(svc.skillRoot, "demo.zip")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected zip file not persisted, stat err=%v", statErr)
	}
}

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

func TestAdminServiceWriteSkillFileContent(t *testing.T) {
	svc := &AdminService{skillRoot: t.TempDir()}

	if err := svc.WriteSkillFileContent("demo/SKILL.md", "name: demo\n"); err != nil {
		t.Fatalf("WriteSkillFileContent failed: %v", err)
	}
	assertFileContent(t, filepath.Join(svc.skillRoot, "demo", "SKILL.md"), "name: demo\n")
}

func TestAdminServiceListSkillsByScope(t *testing.T) {
	systemRoot := t.TempDir()
	workflowRoot := t.TempDir()
	svc := &AdminService{skillRoot: systemRoot, workflowSkillRoot: workflowRoot}

	if err := os.MkdirAll(filepath.Join(systemRoot, "sys"), 0o755); err != nil {
		t.Fatalf("mkdir system dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(systemRoot, "sys", "SKILL.md"), []byte("# system"), 0o644); err != nil {
		t.Fatalf("write system skill: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(workflowRoot, "wf"), 0o755); err != nil {
		t.Fatalf("mkdir workflow dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workflowRoot, "wf", "SKILL.md"), []byte("# workflow"), 0o644); err != nil {
		t.Fatalf("write workflow skill: %v", err)
	}

	sys, err := svc.ListSkillsByScope(context.Background(), "system")
	if err != nil {
		t.Fatalf("ListSkillsByScope(system) failed: %v", err)
	}
	if sys.GetRoot() != systemRoot {
		t.Fatalf("expected system root %q, got %q", systemRoot, sys.GetRoot())
	}
	if len(sys.GetItems()) != 1 || sys.GetItems()[0].GetPath() != "sys/SKILL.md" {
		t.Fatalf("unexpected system items: %#v", sys.GetItems())
	}

	wf, err := svc.ListSkillsByScope(context.Background(), "workflow")
	if err != nil {
		t.Fatalf("ListSkillsByScope(workflow) failed: %v", err)
	}
	if wf.GetRoot() != workflowRoot {
		t.Fatalf("expected workflow root %q, got %q", workflowRoot, wf.GetRoot())
	}
	if len(wf.GetItems()) != 1 || wf.GetItems()[0].GetPath() != "wf/SKILL.md" {
		t.Fatalf("unexpected workflow items: %#v", wf.GetItems())
	}
}

func TestAdminServiceReadWriteSkillFileContentByScope(t *testing.T) {
	systemRoot := t.TempDir()
	workflowRoot := t.TempDir()
	svc := &AdminService{skillRoot: systemRoot, workflowSkillRoot: workflowRoot}

	if err := svc.WriteSkillFileContentByScope("workflow", "pkg/SKILL.md", "workflow skill"); err != nil {
		t.Fatalf("WriteSkillFileContentByScope(workflow) failed: %v", err)
	}
	assertFileContent(t, filepath.Join(workflowRoot, "pkg", "SKILL.md"), "workflow skill")

	if _, err := os.Stat(filepath.Join(systemRoot, "pkg", "SKILL.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no write in system root, stat err=%v", err)
	}

	content, err := svc.ReadSkillFileContentByScope("workflow", "pkg/SKILL.md")
	if err != nil {
		t.Fatalf("ReadSkillFileContentByScope(workflow) failed: %v", err)
	}
	if content != "workflow skill" {
		t.Fatalf("unexpected content: %q", content)
	}
}

func buildZipArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %q: %v", name, err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry %q: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return buf.Bytes()
}

func assertFileContent(t *testing.T, path string, want string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %q: %v", path, err)
	}
	if got := string(data); got != want {
		t.Fatalf("unexpected content for %q, got %q want %q", path, got, want)
	}
}
