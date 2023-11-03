package sql

import (
	"context"
	"sync"

	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/infra/watermill"
)

var (
	initLocker sync.Mutex
)

func initializeSchema(
	ctx context.Context,
	topic string,
	logger watermill.LoggerAdapter,
	db ContextExecutor,
	schemaAdapter SchemaAdapter,
	offsetsAdapter OffsetsAdapter,
) error {
	err := validateTopicName(topic)
	if err != nil {
		return err
	}

	initializingQueries := schemaAdapter.SchemaInitializingQueries(topic)
	if offsetsAdapter != nil {
		initializingQueries = append(initializingQueries, offsetsAdapter.SchemaInitializingQueries(topic)...)
	}

	logger.Info("Initializing subscriber schema", watermill.LogFields{
		"query": initializingQueries,
	})

	// postgres executing create table if not exists DDL is not safe concurrently
	// issue: duplicate key value violates unique constraint "pg_class_relname_nsp_index" (SQLSTATE 23505)
	initLocker.Lock()
	defer initLocker.Unlock()
	for _, q := range initializingQueries {
		_, err := db.ExecContext(ctx, q)
		if err != nil {
			return errors.Wrap(err, "could not initialize schema")
		}
	}

	return nil
}
