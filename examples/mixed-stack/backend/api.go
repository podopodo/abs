package backend

// @ACP O ORDER.CREATE
func CreateOrder(customerID string, total int) error {
	if customerID == "" || total <= 0 {
		return ErrInvalidOrder
	}
	return saveOrder(customerID, total)
}

var ErrInvalidOrder = errorString("invalid order")

type errorString string

func (e errorString) Error() string                { return string(e) }
func saveOrder(customerID string, total int) error { return nil }
