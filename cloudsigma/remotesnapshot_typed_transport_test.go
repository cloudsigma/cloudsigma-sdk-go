package cloudsigma_test

import (
	"context"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func init() { remoteSnapshotRedirectSentinel = sdk.ErrRemoteSnapshotCreateRedirect }
func remoteSnapshotTypedCall(ctx context.Context, c *sdk.Client) (bool, *sdk.Response, error) {
	v, r, e := c.RemoteSnapshots.CreateDefinitions(ctx, &sdk.RemoteSnapshotCreateDefinitionsRequest{Objects: []sdk.RemoteSnapshotCreate{{Drive: &sdk.RemoteSnapshotDriveReference{UUID: "drive-id"}, Name: "remote"}}})
	return v != nil, r, e
}
