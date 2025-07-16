package cases

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/log"
	testUtl "github.com/wfusion/gofusion/test/common/utils"
)

func TestSet(t *testing.T) {
	t.Parallel()
	testingSuite := &Set{Test: new(testUtl.Test)}
	suite.Run(t, testingSuite)
}

type Set struct {
	*testUtl.Test
}

func (t *Set) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Set) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Set) TestInsert() {
	t.Catch(func() {
		set1 := utils.NewSet([]int{1, 2, 3}...)
		set1.Insert(4)
		t.Require().Len(set1.Items(), 4)
		actual := set1.Items()
		sort.Ints(actual)
		t.Require().EqualValues([]int{1, 2, 3, 4}, actual)
	})
}

func (t *Set) TestInteract() {
	t.Catch(func() {
		set1 := utils.NewSet([]int{1, 2, 3}...)
		set2 := utils.NewSet([]int{3, 2, 5}...)
		t.Require().True(set1.IntersectsWith(set2))
		t.Require().True(set2.IntersectsWith(set1))

		expect := []int{2, 3}

		actual := set1.Intersect(set2).Items()
		sort.Ints(actual)
		t.Require().EqualValues(expect, actual)

		actual = set2.Intersect(set1).Items()
		sort.Ints(actual)
		t.Require().EqualValues(expect, actual)
	})
}

func (t *Set) TestUnion() {
	t.Catch(func() {
		set1 := utils.NewSet([]int{1, 2, 3}...)
		set2 := utils.NewSet([]int{3, 2, 5}...)

		expect := []int{1, 2, 3, 5}

		actual := set1.Union(set2).Items()
		sort.Ints(actual)
		t.Require().EqualValues(expect, actual)

		actual = set2.Union(set1).Items()
		sort.Ints(actual)
		t.Require().EqualValues(expect, actual)
	})
}

func (t *Set) TestDiff() {
	t.Catch(func() {
		set1 := utils.NewSet([]int{1, 2, 3}...)
		set2 := utils.NewSet([]int{3, 2, 5}...)

		expect := []int{1}
		actual := set1.Diff(set2).Items()
		sort.Ints(actual)
		t.Require().EqualValues(expect, actual)

		expect = []int{5}
		actual = set2.Diff(set1).Items()
		sort.Ints(actual)
		t.Require().EqualValues(expect, actual)
	})
}
