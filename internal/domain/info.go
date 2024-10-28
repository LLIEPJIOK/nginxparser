package domain

type FileInfo struct {
	Paths            []string
	TotalRequests    int
	AvgResponseSize  int
	ResponseSize95p  int
	FrequentURLs     []URL
	FrequentStatuses []Status
}

func NewFileInfo(
	paths []string,
	totalRequests, avgResponseSize, responseSize95p int,
	frequentURLs []URL,
	frequentStatuses []Status,
) *FileInfo {
	frequentURLsCopy := make([]URL, len(frequentURLs))
	copy(frequentURLsCopy, frequentURLs)

	frequentStatusesCopy := make([]Status, len(frequentStatuses))
	copy(frequentStatusesCopy, frequentStatuses)

	return &FileInfo{
		Paths:            paths,
		TotalRequests:    totalRequests,
		AvgResponseSize:  avgResponseSize,
		ResponseSize95p:  responseSize95p,
		FrequentURLs:     frequentURLsCopy,
		FrequentStatuses: frequentStatusesCopy,
	}
}

type URL struct {
	Name     string
	Quantity int
}

func NewURL(name string, quantity int) URL {
	return URL{
		Name:     name,
		Quantity: quantity,
	}
}

type Status struct {
	Code     int
	Name     string
	Quantity int
}

func NewStatus(code int, name string, quantity int) Status {
	return Status{
		Code:     code,
		Name:     name,
		Quantity: quantity,
	}
}
