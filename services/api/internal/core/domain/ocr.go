package domain

import "slices"

type ExtractedText struct {
	Text     string
	OCRPages []int
}

func MarkReadByOCR(acts []Act, ocrPages []int) []Act {
	out := make([]Act, len(acts))
	for i, a := range acts {
		a.ReadByOCR = a.PageStart > 0 && slices.ContainsFunc(ocrPages, func(p int) bool { return p >= a.PageStart && p <= a.PageEnd })
		out[i] = a
	}
	return out
}
