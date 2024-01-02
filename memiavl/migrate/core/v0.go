package core

import (
	"errors"
	"fmt"
	"time"

	"github.com/cosmos/iavl"
	"github.com/crypto-org-chain/cronos/memiavl"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

func SnapshotCommand() *cobra.Command {
	var (
		dbv0 string
		out  string
	)
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Given an iavl v0 application.db build a memiavl snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			rs, err := NewReadonlyStore(dbv0)
			if err != nil {
				return err
			}

			var (
				storeKeys []string
				version   int64
			)
			for k, ci := range rs.CommitInfoByName() {
				if version != 0 && version != ci.Version {
					return fmt.Errorf("store keys have different versions")
				}
				version = ci.Version
				storeKeys = append(storeKeys, k)
			}

			imp, err := memiavl.NewMultiTreeImporter(out, uint64(version))
			if err != nil {
				return err
			}

			for _, sk := range storeKeys {
				log := logger.With().Str("store", sk).Logger()
				log.Info().Msgf("migrating %s", sk)
				s, err := NewReadonlyStore(dbv0)
				if err != nil {
					return err
				}
				_, tree, err := s.LatestTree(sk)
				if err != nil {
					log.Warn().Err(err).Msgf("skipping %s", sk)
					continue
				}
				err = imp.AddTree(sk)
				if err != nil {
					return err
				}
				exporter, err := tree.Export()
				if err != nil {
					return err
				}
				var (
					count int64
					since = time.Now()
				)
				for {
					count++
					node, err := exporter.Next()
					if err != nil {
						if errors.Is(err, iavl.ErrorExportDone) {
							break
						}
						return err
					}
					imp.AddNode(&memiavl.ExportNode{
						Key:     node.Key,
						Value:   node.Value,
						Version: node.Version,
						Height:  node.Height,
					})
					if count%100_000 == 0 {
						log.Info().Msgf("count=%s dur=%s rate=%s/s",
							humanize.Comma(count),
							time.Since(since),
							humanize.Comma(int64(float64(100_000)/time.Since(since).Seconds())),
						)
						since = time.Now()
					}
				}
			}
			if err = imp.Finalize(); err != nil {
				return err
			}
			if err = imp.Close(); err != nil {
				return err
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&dbv0, "db-v0", "", "Path to the v0 application.db")
	cmd.Flags().StringVar(&out, "out", "", "Path to the output directory")
	if err := cmd.MarkFlagRequired("db-v0"); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired("out"); err != nil {
		panic(err)
	}
	return cmd
}
