package domain

type FileInfo struct {
	Paths             []string
	TotalRequests     int
	AvgResponseSize   int
	ResponseSize95p   int
	AvgResponsePerDay int
	FrequentURLs      []URL
	FrequentStatuses  []Status
	FrequentAddresses []Address
}

func NewFileInfo(
	paths []string,
	totalRequests, avgResponseSize, responseSize95p, avgResponsePerDay int,
	frequentURLs []URL,
	frequentStatuses []Status,
	frequentAddresses []Address,
) *FileInfo {
	return &FileInfo{
		Paths:             paths,
		TotalRequests:     totalRequests,
		AvgResponseSize:   avgResponseSize,
		ResponseSize95p:   responseSize95p,
		AvgResponsePerDay: avgResponsePerDay,
		FrequentURLs:      frequentURLs,
		FrequentStatuses:  frequentStatuses,
		FrequentAddresses: frequentAddresses,
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

type Address struct {
	Name     string
	Quantity int
}

func NewAddress(name string, quantity int) Address {
	return Address{
		Name:     name,
		Quantity: quantity,
	}
}
