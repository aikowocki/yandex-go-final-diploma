package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	clienti18n "github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
	secretuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
	vaultuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/vault"
)

// SecretUpdateCmd — редактирование секрета с оптимистичной блокировкой по версии.
type SecretUpdateCmd struct {
	Vault string `arg:"" help:"Vault name."`
	ID    string `arg:"" help:"Secret id (from 'secret list')."`
}

// Run обновляет секрет login/password, разрешая конфликты версии при необходимости.
func (c *SecretUpdateCmd) Run(auth *authuc.UseCase, vault *vaultuc.UseCase, secret *secretuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}
	vaultID, err := openVaultByName(ctx, vault, c.Vault)
	if err != nil {
		return err
	}

	baseVersion, ok, err := localSecretVersion(ctx, secret, vaultID, c.ID)
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println(l.T("secret_not_found_local"))
		return nil
	}

	input, err := promptLoginPasswordInput(l)
	if err != nil {
		return err
	}

	conflict, err := secret.UpdateLoginPassword(ctx, vaultID, c.ID, baseVersion, input)
	if err != nil {
		return err
	}
	if conflict != nil {
		return resolveGenericConflictInteractive(ctx, secret, l, conflict)
	}
	fmt.Println(l.T("secret_updated"))
	return nil
}

// SecretDeleteCmd — soft-delete секрета с оптимистичной блокировкой по версии.
type SecretDeleteCmd struct {
	Vault string `arg:"" help:"Vault name."`
	ID    string `arg:"" help:"Secret id (from 'secret list')."`
	Yes   bool   `help:"Skip confirmation." short:"y"`
}

// Run выполняет soft-delete секрета, разрешая конфликты версии при необходимости.
func (c *SecretDeleteCmd) Run(auth *authuc.UseCase, vault *vaultuc.UseCase, secret *secretuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}
	vaultID, err := openVaultByName(ctx, vault, c.Vault)
	if err != nil {
		return err
	}

	baseVersion, ok, err := localSecretVersion(ctx, secret, vaultID, c.ID)
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println(l.T("secret_not_found_local"))
		return nil
	}

	if !c.Yes {
		confirmed, err := promptConfirm(l.T("secret_delete_confirm"))
		if err != nil {
			return err
		}
		if !confirmed {
			return nil
		}
	}

	conflict, err := secret.DeleteSecret(ctx, vaultID, c.ID, baseVersion)
	if err != nil {
		return err
	}
	if conflict != nil {
		return resolveGenericConflictInteractive(ctx, secret, l, conflict)
	}
	fmt.Println(l.T("secret_deleted"))
	return nil
}

// SecretSearchCmd — поиск по секретам папки (Tier 2a всегда; Tier 2b — по мере догрузки).
type SecretSearchCmd struct {
	Vault string `arg:"" help:"Vault name."`
	Query string `arg:"" help:"Search query."`
}

// Run выполняет поиск по секретам папки и печатает совпавшие строки.
func (c *SecretSearchCmd) Run(auth *authuc.UseCase, vault *vaultuc.UseCase, secret *secretuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}
	vaultID, err := openVaultByName(ctx, vault, c.Vault)
	if err != nil {
		return err
	}

	res, err := secret.Search(ctx, vaultID, c.Query)
	if err != nil {
		return err
	}
	if res.Incomplete {
		fmt.Println(l.T("search_incomplete"))
	}
	if len(res.Rows) == 0 {
		fmt.Println(l.T("search_no_matches"))
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTITLE\tUSERNAME\tURI")
	for _, r := range res.Rows {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.ID, r.Row.Title, r.Row.Username, r.Row.URI)
	}
	_ = w.Flush()
	return nil
}

// promptLoginPasswordInput запрашивает все поля секрета login/password (общий для add/update).
func promptLoginPasswordInput(l *clienti18n.Localizer) (secretuc.CreateLoginPasswordInput, error) {
	title, err := promptLine(l.T("prompt_title"))
	if err != nil {
		return secretuc.CreateLoginPasswordInput{}, err
	}
	username, err := promptLine(l.T("prompt_username"))
	if err != nil {
		return secretuc.CreateLoginPasswordInput{}, err
	}
	password, err := promptSecret(l.T("prompt_password"))
	if err != nil {
		return secretuc.CreateLoginPasswordInput{}, err
	}
	uri, err := promptLine(l.T("prompt_uri"))
	if err != nil {
		return secretuc.CreateLoginPasswordInput{}, err
	}
	tagsRaw, err := promptLine(l.T("prompt_tags"))
	if err != nil {
		return secretuc.CreateLoginPasswordInput{}, err
	}
	note, err := promptLine(l.T("prompt_note"))
	if err != nil {
		return secretuc.CreateLoginPasswordInput{}, err
	}
	otpCodes, err := promptOTPCodes(l)
	if err != nil {
		return secretuc.CreateLoginPasswordInput{}, err
	}
	return secretuc.CreateLoginPasswordInput{
		Title:    title,
		Username: username,
		Password: string(password),
		URI:      uri,
		Tags:     parseTags(tagsRaw),
		Note:     note,
		OTPCodes: otpCodes,
	}, nil
}

// localSecretVersion находит текущую (локальную) версию секрета по id — это base_version для
// оптимистичной блокировки. Читает из локального кеша (Tier 2a), без сети.
func localSecretVersion(ctx context.Context, secret *secretuc.UseCase, vaultID, id string) (int64, bool, error) {
	rows, err := secret.ListRow(ctx, vaultID)
	if err != nil {
		return 0, false, err
	}
	for _, r := range rows {
		if r.ID == id {
			return r.Version, true, nil
		}
	}
	return 0, false, nil
}

// Секретные типы.
const (
	textSecretType     = 2
	bankCardSecretType = 4
	totpSecretType     = 5
)

func localTypedVersion(ctx context.Context, secret *secretuc.UseCase, vaultID, id string, _ int) (int64, bool, error) {
	return secret.LocalVersion(ctx, id)
}

func resolveGenericConflictInteractive(ctx context.Context, secret *secretuc.UseCase, l *clienti18n.Localizer, conflict *secretuc.GenericConflict) error {
	for conflict != nil {
		printGenericConflict(l, conflict)

		choice, err := promptConflictChoice(l)
		if err != nil {
			return err
		}

		next, err := secret.GenericResolveConflict(ctx, conflict, choice)
		if err != nil {
			return err
		}
		if choice == secretuc.ChoiceServer {
			fmt.Println(l.T("conflict_resolved_server"))
			return nil
		}
		if next == nil {
			fmt.Println(l.T("conflict_resolved_mine"))
			return nil
		}
		conflict = next
	}
	return nil
}

func printGenericConflict(l *clienti18n.Localizer, c *secretuc.GenericConflict) {
	fmt.Println(l.T("conflict_detected"))

	fmt.Println(l.T("conflict_mine_header"))
	if c.IsDelete {
		fmt.Println(l.T("conflict_delete_intent"))
	} else {
		printFieldMap(c.MineRow)
		printFieldMap(c.MineIndex)
		printFieldMap(c.MinePayload)
	}

	fmt.Println(l.T("conflict_server_header"))
	printFieldMap(c.ServerRow)
	printFieldMap(c.ServerIndex)
	printFieldMap(c.ServerPayload)
}

func printFieldMap(m map[string]any) {
	for k, v := range m {
		if v == nil || v == "" {
			continue
		}
		fmt.Printf("%s: %v\n", k, v)
	}
}

// promptConflictChoice требует явного выбора: 'm' (моя) или 's' (серверная). Любой другой ввод
// НЕ трактуется как выбор по умолчанию (обе ветки деструктивны) — переспрашиваем. После
// maxSecretAttempts неверных попыток возвращаем ошибку, не применяя ничего.
func promptConflictChoice(l *clienti18n.Localizer) (secretuc.ConflictChoice, error) {
	for attempt := 0; attempt < maxSecretAttempts; attempt++ {
		answer, err := promptLine(l.T("conflict_choose"))
		if err != nil {
			return "", err
		}
		switch strings.ToLower(strings.TrimSpace(answer)) {
		case "m", "mine":
			return secretuc.ChoiceMine, nil
		case "s", "server":
			return secretuc.ChoiceServer, nil
		default:
			fmt.Println(l.T("conflict_invalid_choice"))
		}
	}
	return "", errInvalidChoice
}
