package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionCmd_Run(t *testing.T) {
	cmd := &VersionCmd{}
	info := &BuildInfo{Version: "1.0.0", Date: "2026-07-17"}
	require.NoError(t, cmd.Run(info))
}
