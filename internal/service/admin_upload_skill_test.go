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
