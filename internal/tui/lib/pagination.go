package lib

import "fmt"

const defaultWindowHeight = 24

type Pagination struct {
	Start      int
	End        int
	PageSize   int
	Page       int
	TotalPages int
}

func Paginate(total, cursor, windowHeight, reservedLines int) Pagination {
	pageSize := windowHeight - reservedLines
	if pageSize < 1 {
		pageSize = 1
	}

	if windowHeight <= 0 {
		pageSize = defaultWindowHeight - reservedLines
		if pageSize < 1 {
			pageSize = 1
		}
	}

	if total <= 0 {
		return Pagination{PageSize: pageSize}
	}

	if cursor < 0 {
		cursor = 0
	}
	if cursor >= total {
		cursor = total - 1
	}

	totalPages := (total + pageSize - 1) / pageSize
	page := cursor / pageSize
	start := page * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}

	return Pagination{
		Start:      start,
		End:        end,
		PageSize:   pageSize,
		Page:       page + 1,
		TotalPages: totalPages,
	}
}

func (p Pagination) StatusLabel(total int) string {
	if total <= 0 || p.TotalPages <= 1 {
		return ""
	}

	return fmt.Sprintf("%d-%d/%d", p.Start+1, p.End, total)
}
