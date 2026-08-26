package agent

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/SuperCoolMinecraftDevs/PhantomMC/internal/manifest"
	"github.com/SuperCoolMinecraftDevs/PhantomMC/internal/msauth"
)

// SignIn runs the device code flow and returns a playable account. There is no
// browser on this machine and nowhere to keep a token, so the user approves on
// a phone and the result lives only as long as the boot.
func SignIn(ctx context.Context, auth manifest.Auth, out io.Writer) (Account, error) {
	cfg := &msauth.Config{ClientID: auth.ClientID}

	code, err := cfg.RequestDeviceCode(ctx)
	if err != nil {
		return Account{}, err
	}

	writeSignInPrompt(out, code)

	token, err := cfg.PollForToken(ctx, code)
	if err != nil {
		return Account{}, err
	}

	profile, err := cfg.Authenticate(ctx, msauth.Endpoints{}, token)
	if err != nil {
		return Account{}, err
	}

	fmt.Fprintf(out, "\nsigned in as %s\n", profile.Name)
	return Account{
		Name:        profile.Name,
		UUID:        profile.UUID,
		AccessToken: profile.AccessToken,
		UserType:    "msa",
	}, nil
}

// writeSignInPrompt draws the code large enough to read from across a room,
// because the person reading it is holding a phone and looking at a television
// or a monitor they are not sitting at.
func writeSignInPrompt(out io.Writer, code *msauth.DeviceCode) {
	border := "+" + repeat("-", 54) + "+"

	fmt.Fprintf(out, "\n%s\n", border)
	fmt.Fprintf(out, "|%s|\n", center("SIGN IN TO MINECRAFT", 54))
	fmt.Fprintf(out, "|%s|\n", repeat(" ", 54))
	fmt.Fprintf(out, "|%s|\n", center("On your phone, open", 54))
	fmt.Fprintf(out, "|%s|\n", center(code.VerificationURI, 54))
	fmt.Fprintf(out, "|%s|\n", repeat(" ", 54))
	fmt.Fprintf(out, "|%s|\n", center("and enter this code", 54))
	fmt.Fprintf(out, "|%s|\n", center(code.UserCode, 54))
	fmt.Fprintf(out, "|%s|\n", repeat(" ", 54))
	expiry := (time.Duration(code.ExpiresIn) * time.Second).Round(time.Minute)
	fmt.Fprintf(out, "|%s|\n", center(fmt.Sprintf("expires in %s", expiry), 54))
	fmt.Fprintf(out, "%s\n\n", border)
}

func repeat(s string, n int) string {
	out := make([]byte, 0, n*len(s))
	for range n {
		out = append(out, s...)
	}
	return string(out)
}

func center(s string, width int) string {
	if len(s) >= width {
		return s
	}
	left := (width - len(s)) / 2
	return repeat(" ", left) + s + repeat(" ", width-len(s)-left)
}
