package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	clienti18n "github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
	secretuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
	vaultuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/vault"
)

// BankCardCmd — группа команд для секретов типа bank_card.
type BankCardCmd struct {
	Add    BankCardAddCmd    `cmd:"" help:"Add a bank card secret to a vault."`
	List   BankCardListCmd   `cmd:"" help:"List bank card secrets of a vault."`
	Show   BankCardShowCmd   `cmd:"" help:"Show a bank card secret's full card (incl. PAN/CVV/PIN)."`
	Update BankCardUpdateCmd `cmd:"" help:"Update a bank card secret (optimistic locking; resolves conflicts)."`
}

type BankCardAddCmd struct {
	Vault string `arg:"" help:"Vault name."`
}

func (c *BankCardAddCmd) Run(auth *authuc.UseCase, vault *vaultuc.UseCase, secret *secretuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}
	vaultID, err := openVaultByName(ctx, vault, c.Vault)
	if err != nil {
		return err
	}

	input, err := promptBankCardInput(l)
	if err != nil {
		return err
	}
	if _, err := secret.CreateBankCard(ctx, vaultID, input); err != nil {
		return err
	}
	fmt.Println(l.T("secret_created"))
	return nil
}

type BankCardListCmd struct {
	Vault string `arg:"" help:"Vault name."`
}

func (c *BankCardListCmd) Run(auth *authuc.UseCase, vault *vaultuc.UseCase, secret *secretuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}
	vaultID, err := openVaultByName(ctx, vault, c.Vault)
	if err != nil {
		return err
	}

	rows, err := secret.ListBankCardRows(ctx, vaultID)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		fmt.Println(l.T("secret_empty"))
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tLAST4")
	for _, r := range rows {
		fmt.Fprintf(w, "%s\t%s\t%s\n", r.ID, r.Row.Title, r.Row.Last4)
	}
	w.Flush()
	return nil
}

type BankCardShowCmd struct {
	Vault string `arg:"" help:"Vault name."`
	ID    string `arg:"" help:"Secret id (from 'secret card list')."`
}

func (c *BankCardShowCmd) Run(auth *authuc.UseCase, vault *vaultuc.UseCase, secret *secretuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}
	vaultID, err := openVaultByName(ctx, vault, c.Vault)
	if err != nil {
		return err
	}

	d, err := secret.GetBankCardDetail(ctx, vaultID, c.ID)
	if err != nil {
		return err
	}
	fmt.Printf("%s: %s\n", l.T("label_title"), d.Row.Title)
	if len(d.Row.Tags) > 0 {
		fmt.Printf("%s: %s\n", l.T("label_tags"), joinTags(d.Row.Tags))
	}
	fmt.Printf("%s: %s\n", l.T("label_bank"), d.Index.Bank)
	fmt.Printf("%s: %s\n", l.T("label_cardholder"), d.Index.Cardholder)
	fmt.Printf("%s: %s\n", l.T("label_brand"), d.Index.Brand)
	fmt.Printf("%s: %s\n", l.T("label_expiry"), d.Index.Expiry)
	if d.Index.Note != "" {
		fmt.Printf("%s: %s\n", l.T("label_note"), d.Index.Note)
	}
	fmt.Printf("%s: %s\n", l.T("label_pan"), d.Payload.PAN)
	fmt.Printf("%s: %s\n", l.T("label_cvv"), d.Payload.CVV)
	if d.Payload.PIN != "" {
		fmt.Printf("%s: %s\n", l.T("label_pin"), d.Payload.PIN)
	}
	printOTPCodes(l, d.Payload.OTPCodes)
	return nil
}

type BankCardUpdateCmd struct {
	Vault string `arg:"" help:"Vault name."`
	ID    string `arg:"" help:"Secret id (from 'secret card list')."`
}

func (c *BankCardUpdateCmd) Run(auth *authuc.UseCase, vault *vaultuc.UseCase, secret *secretuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}
	vaultID, err := openVaultByName(ctx, vault, c.Vault)
	if err != nil {
		return err
	}

	baseVersion, ok, err := localTypedVersion(ctx, secret, vaultID, c.ID, bankCardSecretType)
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println(l.T("secret_not_found_local"))
		return nil
	}

	input, err := promptBankCardInput(l)
	if err != nil {
		return err
	}
	conflict, err := secret.UpdateBankCard(ctx, vaultID, c.ID, baseVersion, input)
	if err != nil {
		return err
	}
	if conflict != nil {
		return resolveGenericConflictInteractive(ctx, secret, l, conflict)
	}
	fmt.Println(l.T("secret_updated"))
	return nil
}

func promptBankCardInput(l *clienti18n.Localizer) (secretuc.CreateBankCardInput, error) {
	title, err := promptLine(l.T("prompt_title"))
	if err != nil {
		return secretuc.CreateBankCardInput{}, err
	}
	pan, err := promptSecret(l.T("prompt_pan"))
	if err != nil {
		return secretuc.CreateBankCardInput{}, err
	}
	cvv, err := promptSecret(l.T("prompt_cvv"))
	if err != nil {
		return secretuc.CreateBankCardInput{}, err
	}
	bank, err := promptLine(l.T("prompt_bank"))
	if err != nil {
		return secretuc.CreateBankCardInput{}, err
	}
	cardholder, err := promptLine(l.T("prompt_cardholder"))
	if err != nil {
		return secretuc.CreateBankCardInput{}, err
	}
	brand, err := promptLine(l.T("prompt_brand"))
	if err != nil {
		return secretuc.CreateBankCardInput{}, err
	}
	expiry, err := promptLine(l.T("prompt_expiry"))
	if err != nil {
		return secretuc.CreateBankCardInput{}, err
	}
	tagsRaw, err := promptLine(l.T("prompt_tags"))
	if err != nil {
		return secretuc.CreateBankCardInput{}, err
	}
	note, err := promptLine(l.T("prompt_note"))
	if err != nil {
		return secretuc.CreateBankCardInput{}, err
	}
	otpCodes, err := promptOTPCodes(l)
	if err != nil {
		return secretuc.CreateBankCardInput{}, err
	}
	return secretuc.CreateBankCardInput{
		Title: title, PAN: string(pan), CVV: string(cvv), Bank: bank,
		Cardholder: cardholder, Brand: brand, Expiry: expiry,
		Tags: parseTags(tagsRaw), Note: note, OTPCodes: otpCodes,
	}, nil
}
