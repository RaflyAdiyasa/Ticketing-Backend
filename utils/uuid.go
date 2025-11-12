package utils

import (
	"fmt"
	"strings"
	"github.com/google/uuid"
)

func GeneratePrefixedUUID(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, uuid.New().String())
}

func GenerateUserID(role string) string {
	return GeneratePrefixedUUID(role)
}

func GenerateEventID() string {
	return GeneratePrefixedUUID("event")
}

func GenerateTicketCategoryID() string {
	return GeneratePrefixedUUID("tcat")
}

func GenerateTicketID() string {
	return GeneratePrefixedUUID("ticket")
}

func GenerateTicketCode() string {
	uuidStr := uuid.New().String()
	cleanUUID := strings.ReplaceAll(uuidStr, "-", "")
	code := fmt.Sprintf("tix%s", cleanUUID[:10])
	return code
}

func GenerateCartID() string {
	return GeneratePrefixedUUID("cart")
}

func GenerateTransactionID() string {
	return GeneratePrefixedUUID("trans")
}

func GenerateTransactionDetailID() string {
	return GeneratePrefixedUUID("tdet")
}

func GenerateRandomName() string {
	return GeneratePrefixedUUID("name")
}

func GenerateRandomEmail() string {
	return fmt.Sprintf("%s@gmail.com", uuid.New().String())
}
