package biz

import (
	"encoding/base64"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/schema"
)

const (
	maxAttachmentTextRunes   = 120000
	maxAttachmentBase64Chars = 350000
	maxLegacyFallbackBytes   = 480000
)

// 与 eino-ext acl/openai 中 InputAudio 支持的 MIME 对齐；不在此集合内的音频退回为纯文本附件块。
var harnessAudioMIMEForMultimodal = map[string]struct{}{
	"audio/wav":      {},
	"audio/vnd.wav":  {},
	"audio/vnd.wave": {},
	"audio/wave":     {},
	"audio/x-pn-wav": {},
	"audio/mpeg":     {},
	"audio/mp3":      {},
	"audio/x-wav":    {},
	"audio/mpeg3":    {},
	"audio/x-mpeg-3": {},
}

type HarnessMultimodalOptions struct {
	DisableGenericFilePart bool
}

func legacyAttachmentBlock(fn, mime, txtIn, b64In string) string {
	txt := strings.TrimSpace(txtIn)
	b64 := strings.TrimSpace(b64In)
	var b strings.Builder
	b.WriteString("\n\n---\n【附件】 ")
	b.WriteString(fn)
	if mime != "" {
		b.WriteString(" • ")
		b.WriteString(mime)
	}
	b.WriteString("\n")
	if txt != "" {
		b.WriteString(txt)
		b.WriteString("\n")
	}
	if b64 != "" {
		b.WriteString("(以下为文件原始字节 Base64；当前通道仅对图/音/视频走多模态；其它类型以文本形式附加)\n")
		b.WriteString(b64)
		b.WriteString("\n")
	}
	if txt == "" && b64 == "" {
		b.WriteString("(附件无文本或 base64 内容)\n")
	}
	out := b.String()
	if len(out) > maxLegacyFallbackBytes {
		return out[:maxLegacyFallbackBytes] + "\n…（附件块过长已截断）"
	}
	return out
}

// SniffBinaryMIME 根据文件头推断 MIME（对话层拉取外链等场景可复用）。
func SniffBinaryMIME(p []byte) string {
	return sniffMIMEFromBinary(p)
}

// sniffMIMEFromBinary 根据文件头推断 MIME（用于浏览器未填 file.type 或为 application/octet-stream 的情形）。
func sniffMIMEFromBinary(p []byte) string {
	n := len(p)
	if n < 4 {
		return ""
	}
	if n >= 8 && p[0] == 0x89 && p[1] == 'P' && p[2] == 'N' && p[3] == 'G' {
		return "image/png"
	}
	if n >= 3 && p[0] == 0xFF && p[1] == 0xD8 && p[2] == 0xFF {
		return "image/jpeg"
	}
	if n >= 2 && p[0] == 0x42 && p[1] == 0x4D {
		return "image/bmp"
	}
	if n >= 6 && (string(p[0:6]) == "GIF87a" || string(p[0:6]) == "GIF89a") {
		return "image/gif"
	}
	if n >= 12 && string(p[0:4]) == "RIFF" && string(p[8:12]) == "WEBP" {
		return "image/webp"
	}
	if n >= 4 && p[0] == 0x1A && p[1] == 0x45 && p[2] == 0xDF && p[3] == 0xA3 {
		return "video/webm"
	}
	lim := n
	if lim > 4096 {
		lim = 4096
	}
	for i := 0; i <= lim-4; i++ {
		if p[i] == 'f' && p[i+1] == 't' && p[i+2] == 'y' && p[i+3] == 'p' {
			return "video/mp4"
		}
	}
	return ""
}

func mimeFromFilename(fn string) string {
	switch strings.ToLower(filepath.Ext(fn)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg", ".jpe":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".bmp":
		return "image/bmp"
	case ".mp4", ".m4v":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	default:
		return ""
	}
}

// resolveAttachmentMIME 合并客户端声明、魔数与扩展名，避免 image/* 被误判为 octet-stream 导致仅走纯文本 Base64（模型无法视觉理解）。
func resolveAttachmentMIME(filename, declared string, b64 string) string {
	d := strings.TrimSpace(strings.ToLower(declared))
	if strings.HasPrefix(d, "image/") || strings.HasPrefix(d, "video/") || strings.HasPrefix(d, "audio/") {
		return d
	}
	raw, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(strings.TrimSpace(b64), "\n", ""))
	if err == nil && len(raw) > 0 {
		if m := sniffMIMEFromBinary(raw); m != "" {
			return m
		}
	}
	if m := mimeFromFilename(filename); m != "" {
		return m
	}
	return d
}

// buildHarnessInputParts 将用户输入与附件转为 Eino UserInputMultiContent（图片/音视频走 OpenAI 兼容多模态；其余附件退回纯文本块）。
func buildHarnessInputParts(userText string, attachments []HarnessAttachment) []schema.MessageInputPart {
	return buildHarnessInputPartsWithOptions(userText, attachments, HarnessMultimodalOptions{})
}

func buildHarnessInputPartsWithOptions(userText string, attachments []HarnessAttachment, opts HarnessMultimodalOptions) []schema.MessageInputPart {
	userText = strings.TrimSpace(userText)
	var parts []schema.MessageInputPart
	var legacyBuf strings.Builder

	if userText != "" {
		parts = append(parts, schema.MessageInputPart{
			Type: schema.ChatMessagePartTypeText,
			Text: userText,
		})
	}

	for _, a := range attachments {
		fn := strings.TrimSpace(a.Filename)
		if fn == "" {
			fn = "(未命名)"
		}
		mime := strings.TrimSpace(strings.ToLower(a.MimeType))
		txt := strings.TrimSpace(a.Text)
		b64 := strings.TrimSpace(strings.ReplaceAll(a.ContentBase64, "\n", ""))

		if b64 == "" && txt != "" {
			runes := []rune(txt)
			if len(runes) > maxAttachmentTextRunes {
				txt = string(runes[:maxAttachmentTextRunes]) + "\n…（本附件文本已截断）"
			}
			parts = append(parts, schema.MessageInputPart{
				Type: schema.ChatMessagePartTypeText,
				Text: legacyAttachmentBlock(fn, mime, txt, ""),
			})
			continue
		}
		if b64 == "" && txt == "" {
			legacyBuf.WriteString(legacyAttachmentBlock(fn, mime, "", ""))
			continue
		}

		if len(b64) > maxAttachmentBase64Chars {
			b64 = b64[:maxAttachmentBase64Chars] + "\n…（base64 过长已截断）"
		}

		data := b64
		effective := resolveAttachmentMIME(fn, mime, b64)
		switch {
		case strings.HasPrefix(effective, "image/"):
			im := effective
			if im == "" {
				im = "image/png"
			}
			parts = append(parts, schema.MessageInputPart{
				Type: schema.ChatMessagePartTypeImageURL,
				Image: &schema.MessageInputImage{
					MessagePartCommon: schema.MessagePartCommon{
						Base64Data: &data,
						MIMEType:   im,
					},
					Detail: schema.ImageURLDetailAuto,
				},
			})
		case strings.HasPrefix(effective, "video/"):
			vm := effective
			if vm == "" {
				vm = "video/mp4"
			}
			parts = append(parts, schema.MessageInputPart{
				Type: schema.ChatMessagePartTypeVideoURL,
				Video: &schema.MessageInputVideo{
					MessagePartCommon: schema.MessagePartCommon{
						Base64Data: &data,
						MIMEType:   vm,
					},
				},
			})
		case strings.HasPrefix(effective, "audio/"):
			if _, ok := harnessAudioMIMEForMultimodal[effective]; ok {
				parts = append(parts, schema.MessageInputPart{
					Type: schema.ChatMessagePartTypeAudioURL,
					Audio: &schema.MessageInputAudio{
						MessagePartCommon: schema.MessagePartCommon{
							Base64Data: &data,
							MIMEType:   effective,
						},
					},
				})
			} else {
				legacyBuf.WriteString(legacyAttachmentBlock(fn, mime, txt, b64))
			}
		default:
			if opts.DisableGenericFilePart {
				legacyBuf.WriteString(legacyAttachmentBlock(fn, mime, txt, b64))
				continue
			}
			fm := effective
			if fm == "" {
				fm = mime
			}
			parts = append(parts, schema.MessageInputPart{
				Type: schema.ChatMessagePartTypeFileURL,
				File: &schema.MessageInputFile{
					MessagePartCommon: schema.MessagePartCommon{
						Base64Data: &data,
						MIMEType:   fm,
					},
					Name: fn,
				},
			})
			if txt != "" {
				parts = append(parts, schema.MessageInputPart{
					Type: schema.ChatMessagePartTypeText,
					Text: legacyAttachmentBlock(fn, mime, txt, ""),
				})
			}
		}
	}

	if legacyBuf.Len() > 0 {
		s := legacyBuf.String()
		if len(s) > maxLegacyFallbackBytes {
			s = s[:maxLegacyFallbackBytes] + "\n…（附件汇总过长已截断）"
		}
		parts = append(parts, schema.MessageInputPart{
			Type: schema.ChatMessagePartTypeText,
			Text: s,
		})
	}

	return parts
}

func lastUserMessageFromParts(parts []schema.MessageInputPart) *schema.Message {
	if len(parts) == 0 {
		return schema.UserMessage("")
	}
	if len(parts) == 1 && parts[0].Type == schema.ChatMessagePartTypeText {
		return schema.UserMessage(parts[0].Text)
	}
	return &schema.Message{
		Role:                  schema.User,
		UserInputMultiContent: parts,
	}
}
