package kv

import (
	"context"

	bolt "go.etcd.io/bbolt"
)

// SaveGenesisValidatorsRoot saves the genesis validator root to db.
func (s *Store) SaveGenesisValidatorsRoot(ctx context.Context, genValRoot []byte) error {
	err := s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(genesisInfoBucket)
		return bkt.Put(genesisValidatorsRootKey, genValRoot)
	})
	return err
}

// GenesisValidatorsRoot retrieves the genesis validator root from db.
func (s *Store) GenesisValidatorsRoot(ctx context.Context) ([]byte, error) {
	var genValRoot []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(genesisInfoBucket)
		enc := bkt.Get(genesisValidatorsRootKey)
		if len(enc) == 0 {
			return nil
		}
		genValRoot = enc
		return nil
	})
	return genValRoot, err
}
