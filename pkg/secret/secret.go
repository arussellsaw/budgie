package secret

import (
	"context"
	"encoding/base64"
	"fmt"

	kms "cloud.google.com/go/kms/apiv1"
	secretmanager "cloud.google.com/go/secretmanager/apiv1beta1"

	"github.com/arussellsaw/budgie/pkg/util"

	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
	secrets "google.golang.org/genproto/googleapis/cloud/secretmanager/v1beta1"
)

var (
	projectID  = "youneedaspreadsheet"
	locationID = "global"
)

type Secret struct {
	value string
}

func (s *Secret) String() string {
	return "********"
}

func (s *Secret) Secret() string {
	return s.value
}

func Get(ctx context.Context, name string) (string, error) {
	sm, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", err
	}
	defer sm.Close()

	res, err := sm.AccessSecretVersion(
		ctx,
		&secrets.AccessSecretVersionRequest{Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, name)},
	)
	if err != nil {
		return "", err
	}

	return string(res.GetPayload().GetData()), nil
}

func Encrypt(ctx context.Context, plaintext []byte) (string, string, error) {
	client, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return "", "", err
	}
	defer client.Close()

	path := "projects/" + util.Project() + "/locations/global/keyRings/oauth/cryptoKeys/access_tokens/cryptoKeyVersions/1"
	res, err := client.Encrypt(ctx, &kmspb.EncryptRequest{
		Name:      path,
		Plaintext: []byte(plaintext),
	})
	if err != nil {
		return "", "", err
	}
	b64Ciphertext := base64.StdEncoding.EncodeToString(res.GetCiphertext())

	return b64Ciphertext, res.GetName(), nil
}

func Decrypt(ctx context.Context, ciphertext, keyName string) ([]byte, error) {
	client, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	plainCiphertext, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, err
	}

	keyName = "projects/" + util.Project() + "/locations/global/keyRings/oauth/cryptoKeys/access_tokens"

	res, err := client.Decrypt(ctx, &kmspb.DecryptRequest{
		Name:       keyName,
		Ciphertext: plainCiphertext,
	})
	if err != nil {
		return nil, err
	}
	return res.GetPlaintext(), nil
}
