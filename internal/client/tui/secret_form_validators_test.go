package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateDigits(t *testing.T) {
	assert.NoError(t, validateDigits("12345"))
	assert.NoError(t, validateDigits(""))
	assert.Error(t, validateDigits("123a"))
	assert.Error(t, validateDigits("1 2"))
}

func TestValidateExpiry(t *testing.T) {
	assert.NoError(t, validateExpiry("12/25"))
	assert.NoError(t, validateExpiry("1225"))
	assert.NoError(t, validateExpiry(""))
	assert.Error(t, validateExpiry("12/2/5"))
	assert.Error(t, validateExpiry("ab/cd"))
}

func TestValidatePAN(t *testing.T) {
	assert.NoError(t, validatePAN("1234 5678 9012 3456"))
	assert.NoError(t, validatePAN("1234567890123456"))
	assert.Error(t, validatePAN("1234-5678"))
	assert.Error(t, validatePAN("abcd"))
}

func TestFormatExpiry(t *testing.T) {
	assert.Equal(t, "1", formatExpiry("1"))
	assert.Equal(t, "12", formatExpiry("12"))
	assert.Equal(t, "12/2", formatExpiry("122"))
	assert.Equal(t, "12/25", formatExpiry("1225"))
	assert.Equal(t, "12/25", formatExpiry("12/25/99"), "extra digits truncated to 4")
}

func TestFormatPAN(t *testing.T) {
	assert.Equal(t, "1234", formatPAN("1234"))
	assert.Equal(t, "1234 5678", formatPAN("12345678"))
	assert.Equal(t, "1234 5678 9012 3456", formatPAN("1234567890123456"))
	assert.Equal(t, "1234 5678 9012 3456", formatPAN("1234-5678-9012-3456-9999"), "extra digits truncated to 16")
}

func TestOnlyDigits(t *testing.T) {
	assert.Equal(t, "12345", onlyDigits("1a2b3c4d5"))
	assert.Equal(t, "", onlyDigits("abc"))
}

func TestLuhnValid(t *testing.T) {
	// 4532015112830366 — известный валидный тестовый номер Visa (проходит Luhn).
	assert.True(t, luhnValid("4532015112830366"))
	assert.False(t, luhnValid("4532015112830367"))
	assert.False(t, luhnValid("123")) // too short
	assert.False(t, luhnValid(""))
}

func TestLuhnCheckDigit(t *testing.T) {
	// Для "453201511283036" контрольная цифра должна быть 6.
	assert.Equal(t, 6, luhnCheckDigit("453201511283036"))
}

func TestDetectCardBrand(t *testing.T) {
	assert.Equal(t, "Visa", detectCardBrand("4532015112830366"))
	assert.Equal(t, "Mastercard", detectCardBrand("5112345678901234"))
	assert.Equal(t, "Mastercard", detectCardBrand("2223000048400011"))
	assert.Equal(t, "Mir", detectCardBrand("2200123456789012"))
	assert.Equal(t, "AmEx", detectCardBrand("341234567890123"))
	assert.Equal(t, "JCB", detectCardBrand("3530123456789012"))
	assert.Equal(t, "UnionPay", detectCardBrand("6212345678901234"))
	assert.Equal(t, "", detectCardBrand("999"))
	assert.Equal(t, "", detectCardBrand("9999999999999999"))
}

func TestResolveFilePath(t *testing.T) {
	assert.Equal(t, "", resolveFilePath(""))
	assert.Equal(t, "/tmp/foo.txt", resolveFilePath("/tmp/foo.txt"))
	assert.Equal(t, "./relative/path.txt", resolveFilePath("./relative/path.txt"))
}

func TestResolveFilePath_FileURI(t *testing.T) {
	got := resolveFilePath("file:///tmp/foo.txt")
	assert.Equal(t, "/tmp/foo.txt", got)
}

func TestFindDuplicateCustomKey(t *testing.T) {
	mk := func(k string) kvPair {
		ti := textinput.New()
		ti.SetValue(k)
		return kvPair{key: ti}
	}
	assert.Equal(t, "", findDuplicateCustomKey(nil))
	assert.Equal(t, "", findDuplicateCustomKey([]kvPair{mk("a"), mk("b")}))
	assert.Equal(t, "a", findDuplicateCustomKey([]kvPair{mk("a"), mk("a")}))
	assert.Equal(t, "", findDuplicateCustomKey([]kvPair{mk(""), mk("")}), "empty keys ignored")
}

func TestFindDuplicateOTPCode(t *testing.T) {
	mk := func(c string) otpItem {
		ti := textinput.New()
		ti.SetValue(c)
		return otpItem{code: ti}
	}
	require.Equal(t, "", findDuplicateOTPCode(nil))
	assert.Equal(t, "", findDuplicateOTPCode([]otpItem{mk("111111"), mk("222222")}))
	assert.Equal(t, "111111", findDuplicateOTPCode([]otpItem{mk("111111"), mk("111111")}))
}
