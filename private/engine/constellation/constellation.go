package constellation

import (
	"fmt"

	"github.com/ethereum/go-ethereum/private/engine"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/private/cache"

	gocache "github.com/patrickmn/go-cache"
)

type constellation struct {
	node *Client
	c    *gocache.Cache
}

func Is(ptm interface{}) bool {
	_, ok := ptm.(*constellation)
	return ok
}

func New(client *engine.Client) *constellation {
	return &constellation{
		node: &Client{
			httpClient: client.HttpClient,
		},
		c: gocache.New(cache.DefaultExpiration, cache.CleanupInterval),
	}
}

func (g *constellation) Send(data []byte, from string, to []string, extra *engine.ExtraMetadata) (common.EncryptedPayloadHash, error) {
	out, err := g.node.SendPayload(data, from, to, extra.ACHashes, extra.ACMerkleRoot)
	if err != nil {
		return common.EncryptedPayloadHash{}, err
	}
	cacheKey := string(out.Bytes())
	g.c.Set(cacheKey, cache.PrivateCacheItem{
		Payload: data,
		Extra:   *extra,
	}, cache.DefaultExpiration)
	return out, nil
}

func (g *constellation) SendSignedTx(data common.EncryptedPayloadHash, to []string, extra *engine.ExtraMetadata) (out []byte, err error) {
	return nil, engine.ErrPrivateTxManagerNotSupported
}

func (g *constellation) ReceiveRaw(data common.EncryptedPayloadHash) ([]byte, *engine.ExtraMetadata, error) {
	return nil, nil, engine.ErrPrivateTxManagerNotSupported
}

func (g *constellation) Receive(data common.EncryptedPayloadHash) ([]byte, *engine.ExtraMetadata, error) {
	if common.EmptyEncryptedPayloadHash(data) {
		return data.Bytes(), nil, nil
	}
	// Ignore this error since not being a recipient of
	// a payload isn't an error.
	// TODO: Return an error if it's anything OTHER than
	// 'you are not a recipient.'
	cacheKey := string(data.Bytes())
	x, found := g.c.Get(cacheKey)
	if found {
		cacheItem, ok := x.(cache.PrivateCacheItem)
		if !ok {
			return nil, nil, fmt.Errorf("unknown cache item. expected type PrivateCacheItem")
		}
		return cacheItem.Payload, &cacheItem.Extra, nil
	}
	privatePayload, acHashes, acMerkleRoot, err := g.node.ReceivePayload(data)
	if nil != err {
		return nil, nil, err
	}
	extra := engine.ExtraMetadata{
		ACHashes:     acHashes,
		ACMerkleRoot: acMerkleRoot,
	}
	g.c.Set(cacheKey, cache.PrivateCacheItem{
		Payload: privatePayload,
		Extra:   extra,
	}, cache.DefaultExpiration)
	return privatePayload, &extra, nil
}

func (g *constellation) Name() string {
	return "Constellation"
}
