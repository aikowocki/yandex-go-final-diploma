package secret

import (
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
)

// GenerateTOTPCode генерирует текущий TOTP-код из расшифрованного payload секрета.
// Секрет никогда не покидает клиент.
func GenerateTOTPCode(p secretcontent.TOTPPayload) (string, error) {
	opts := totp.ValidateOpts{
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	}
	if p.Digits > 0 {
		switch p.Digits {
		case 8:
			opts.Digits = otp.DigitsEight
		default:
			opts.Digits = otp.DigitsSix
		}
	}
	if p.Period > 0 {
		opts.Period = uint(p.Period)
	} else {
		opts.Period = 30
	}
	if algo := parseAlgo(p.Algo); algo != 0 {
		opts.Algorithm = algo
	}
	return totp.GenerateCodeCustom(p.Secret, time.Now(), opts)
}

func parseAlgo(algo string) otp.Algorithm {
	switch strings.ToUpper(strings.TrimSpace(algo)) {
	case "SHA256":
		return otp.AlgorithmSHA256
	case "SHA512":
		return otp.AlgorithmSHA512
	case "SHA1", "":
		return otp.AlgorithmSHA1
	default:
		return 0
	}
}

// ParseOTPAuthURI парсит ссылку/QR-контент вида otpauth://totp/Issuer:Account?secret=...&issuer=...
// (стандартный формат, тот же что генерируют Google Authenticator/1Password и т.п.). Позволяет
// пользователю вставить скопированную ссылку (или текст, распознанный из QR) вместо ручного ввода
// каждого поля.
func ParseOTPAuthURI(uri string) (CreateTOTPInput, error) {
	key, err := otp.NewKeyFromURL(strings.TrimSpace(uri))
	if err != nil {
		return CreateTOTPInput{}, ErrInvalidOTPAuthURI
	}
	if key.Secret() == "" {
		return CreateTOTPInput{}, ErrInvalidOTPAuthURI
	}

	digits := 6
	if d := key.Digits(); d.Length() > 0 {
		digits = d.Length()
	}
	period := 30
	if p := key.Period(); p > 0 {
		period = int(p)
	}

	issuer := key.Issuer()
	account := key.AccountName()
	title := issuer
	if title == "" {
		title = account
	}

	return CreateTOTPInput{
		Title:   title,
		Issuer:  issuer,
		Account: account,
		Secret:  key.Secret(),
		Algo:    key.Algorithm().String(),
		Digits:  digits,
		Period:  period,
	}, nil
}
