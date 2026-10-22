package chunk

import "strings"

// Chunk adalah satu potongan dokumen yang siap di-embed.
type Chunk struct {
	Heading string // judul section, mis. "Jam Operasional"
	Content string // teks yang di-embed & nanti dikirim ke LLM
}

type section struct {
	heading string
	lines   []string
}

// Split memotong dokumen markdown per section "## ".
// Section yang lebih panjang dari maxChars dipecah per paragraf,
// dengan paragraf terakhir diulang di potongan berikutnya (overlap).
func Split(markdown string, maxChars int) []Chunk {
	var title string
	var sections []section
	current := section{}

	for _, line := range strings.Split(markdown, "\n") {
		switch {
		case strings.HasPrefix(line, "# "):
			title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
		case strings.HasPrefix(line, "## "):
			sections = append(sections, current)
			current = section{heading: strings.TrimSpace(strings.TrimPrefix(line, "## "))}
		default:
			current.lines = append(current.lines, line)
		}
	}
	sections = append(sections, current)

	var chunks []Chunk
	for _, s := range sections {
		body := strings.TrimSpace(strings.Join(s.lines, "\n"))
		if body == "" {
			continue // lewati section kosong
		}
		for _, part := range splitLong(body, maxChars) {
			chunks = append(chunks, Chunk{
				Heading: s.heading,
				Content: header(title, s.heading) + part,
			})
		}
	}
	return chunks
}

// header menambahkan konteks judul supaya potongan tetap bermakna saat berdiri sendiri.
func header(title, heading string) string {
	switch {
	case title != "" && heading != "":
		return title + " - " + heading + "\n\n"
	case title != "":
		return title + "\n\n"
	case heading != "":
		return heading + "\n\n"
	}
	return ""
}

// splitLong memecah teks panjang per paragraf dengan overlap satu paragraf.
func splitLong(body string, maxChars int) []string {
	if len(body) <= maxChars {
		return []string{body}
	}

	var parts, buf []string
	size := 0
	for _, p := range strings.Split(body, "\n\n") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if size > 0 && size+len(p) > maxChars {
			parts = append(parts, strings.Join(buf, "\n\n"))
			last := buf[len(buf)-1]
			buf, size = nil, 0
			if len(last)+len(p) <= maxChars { // overlap hanya kalau muat
				buf, size = []string{last}, len(last)
			}
		}
		buf = append(buf, p)
		size += len(p)
	}
	if len(buf) > 0 {
		parts = append(parts, strings.Join(buf, "\n\n"))
	}
	return parts
}

