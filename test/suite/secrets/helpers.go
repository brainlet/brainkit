package secrets

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit/modules/secrets/secretmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/stretchr/testify/require"
)

func callSecretSet(t *testing.T, rt sdk.CallerRuntime, ctx context.Context, msg secretmsg.SecretsSetMsg) secretmsg.SecretsSetResp {
	t.Helper()
	resp, err := secretmsg.CallSecretsSet(rt, ctx, msg, sdk.WithCallTimeout(5*time.Second))
	require.NoError(t, err)
	return resp
}

func callSecretGet(t *testing.T, rt sdk.CallerRuntime, ctx context.Context, msg secretmsg.SecretsGetMsg) secretmsg.SecretsGetResp {
	t.Helper()
	resp, err := secretmsg.CallSecretsGet(rt, ctx, msg, sdk.WithCallTimeout(5*time.Second))
	require.NoError(t, err)
	return resp
}

func callSecretDelete(t *testing.T, rt sdk.CallerRuntime, ctx context.Context, msg secretmsg.SecretsDeleteMsg) secretmsg.SecretsDeleteResp {
	t.Helper()
	resp, err := secretmsg.CallSecretsDelete(rt, ctx, msg, sdk.WithCallTimeout(5*time.Second))
	require.NoError(t, err)
	return resp
}

func callSecretList(t *testing.T, rt sdk.CallerRuntime, ctx context.Context, msg secretmsg.SecretsListMsg) secretmsg.SecretsListResp {
	t.Helper()
	resp, err := secretmsg.CallSecretsList(rt, ctx, msg, sdk.WithCallTimeout(5*time.Second))
	require.NoError(t, err)
	return resp
}

func callSecretRotate(t *testing.T, rt sdk.CallerRuntime, ctx context.Context, msg secretmsg.SecretsRotateMsg) secretmsg.SecretsRotateResp {
	t.Helper()
	resp, err := secretmsg.CallSecretsRotate(rt, ctx, msg, sdk.WithCallTimeout(5*time.Second))
	require.NoError(t, err)
	return resp
}
