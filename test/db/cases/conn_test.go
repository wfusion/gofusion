package cases

import (
	"context"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/db"
	"github.com/wfusion/gofusion/log"

	testDB "github.com/wfusion/gofusion/test/db"
)

func TestConn(t *testing.T) {
	testingSuite := &Conn{Test: new(testDB.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Conn struct {
	*testDB.Test
}

func (t *Conn) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Conn) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Conn) TestMysql() {
	t.testDefault(nameMysqlRead, nameMysqlWrite)
}

func (t *Conn) TestPostgres() {
	t.testDefault(namePostgres, namePostgres)
}

func (t *Conn) TestOpengauss() {
	t.testDefault(nameOpenGauss, nameOpenGauss)
}

func (t *Conn) TestSqlserver() {
	t.testDefault(nameSqlserver, nameSqlserver)
}

func (t *Conn) testDefault(read, write string) {
	t.Catch(func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		tx := db.Use(ctx, write, db.AppName(t.AppName()))
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for range ticker.C {
				connPool, err := tx.GetProxy().DB()
				if err != nil {
					log.Warn(ctx, "get conn pool failed: %s", err)
					continue
				}
				s := connPool.Stats()
				log.Info(ctx, "inuse(%v) idle(%v) open_conns(%v) pending(%v)",
					s.InUse, s.Idle, s.OpenConnections, s.WaitCount)
			}
		}()

		canExecChan := make(chan struct{}, 20)
		go func() {
			ticker := time.NewTicker(time.Millisecond)
			defer ticker.Stop()
			defer close(canExecChan)
			for {
				select {
				case <-ctx.Done():
				case <-ticker.C:
					canExecChan <- struct{}{}
				}
			}
		}()

		wg := new(sync.WaitGroup)
		t.NoError(tx.Migrator().AutoMigrate(new(modelWithData)))
		defer func() {
			t.NoError(tx.Migrator().DropTable(new(modelWithData)))
		}()

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				// jitter within 500ms ~ 1500ms
				time.Sleep(500*time.Millisecond + time.Duration(rand.Float64()*float64(time.Second)))

				<-canExecChan
				mwd1 := &modelWithData{Name: "az1"}
				tx.Create(mwd1)
				//tx.Create(mwd1)

				// jitter within 500ms ~ 1500ms
				<-canExecChan
				time.Sleep(500*time.Millisecond + time.Duration(rand.Float64()*float64(time.Second)))
				tx.Where("name = 'az1'").Delete(mwd1)
				//tx.Where("name = 'az1'").Delete(mwd1)
			}()
		}
		wg.Wait()
	})
}
