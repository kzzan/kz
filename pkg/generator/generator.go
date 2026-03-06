package generator

import (
	"strings"
)

func appendAfterLastMatch(content, keyword, newLine string) string {
	lines := strings.Split(content, "\n")
	lastIdx := -1
	for i, line := range lines {
		if strings.Contains(line, keyword) {
			lastIdx = i
		}
	}
	if lastIdx == -1 {
		return content
	}
	result := make([]string, 0, len(lines)+1)
	result = append(result, lines[:lastIdx+1]...)
	result = append(result, newLine)
	result = append(result, lines[lastIdx+1:]...)
	return strings.Join(result, "\n")
}

func ensureImport(content, importLine string) string {
	if strings.Contains(content, importLine) {
		return content
	}
	importClose := strings.Index(content, "\n)")
	if importClose == -1 {
		return content
	}
	return content[:importClose] + "\n\t" + importLine + content[importClose:]
}

func appendBeforeLastFuncClose(content, funcSignature, newLine string) string {
	funcIdx := strings.Index(content, funcSignature)
	if funcIdx == -1 {
		return content
	}
	braceStart := strings.Index(content[funcIdx:], "{")
	if braceStart == -1 {
		return content
	}
	absStart := funcIdx + braceStart

	depth := 0
	closeIdx := -1
	for i := absStart; i < len(content); i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				closeIdx = i
				goto found
			}
		}
	}
found:
	if closeIdx == -1 {
		return content
	}
	return content[:closeIdx] + newLine + "\n" + content[closeIdx:]
}
