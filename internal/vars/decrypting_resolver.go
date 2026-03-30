package vars

import (
	"log"

	"dops/internal/crypto"
	"dops/internal/domain"
)

type DecryptingVarResolver struct {
	inner VarResolver
	encrypter crypto.Encrypter
}

func NewDecryptingResolver(inner VarResolver, encrypter crypto.Encrypter) *DecryptingVarResolver {
	return &DecryptingVarResolver{inner: inner, encrypter: encrypter}
}

func (r *DecryptingVarResolver) Resolve(cfg *domain.Config, catalogName, runbookName string, params []domain.Parameter) map[string]string {
	resolved := r.inner.Resolve(cfg, catalogName, runbookName, params)

	for k, v := range resolved {
		if crypto.IsEncrypted(v) {
			decrypted, err := r.encrypter.Decrypt(v)
			if err != nil {
				log.Printf("warning: failed to decrypt variable %q: %v", k, err)
				resolved[k] = "DECRYPTION_FAILED"
				continue
			}
			resolved[k] = decrypted
		}
	}

	return resolved
}

var _ VarResolver = (*DecryptingVarResolver)(nil)
