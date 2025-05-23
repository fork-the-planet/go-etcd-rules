package lock

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_nestedLocker_Lock(t *testing.T) {
	// Set up mock data for mock functions
	type testLock struct {
		RuleLock
		val string // Just something to compare.
	}
	var ownUnlockCalled bool
	testOwnLock := testLock{
		RuleLock: FuncMockLock{
			UnlockF: func(_ ...Option) error {
				ownUnlockCalled = true
				return nil
			},
		},
		val: "own",
	}
	testNestedLock := testLock{
		val: "nested",
	}

	ownLockErr := errors.New("own lock")
	nestedLockErr := errors.New("nested lock")

	testCases := []struct {
		name string

		nestedCalled    bool
		ownUnlockCalled bool

		err           error
		ownLockErr    error
		nestedLockErr error
	}{
		{
			name:         "ok",
			nestedCalled: true,
		},
		{
			name:       "own_error",
			ownLockErr: ownLockErr,
			err:        ownLockErr,
		},
		{
			name:            "nested_error",
			nestedCalled:    true,
			ownUnlockCalled: true,
			nestedLockErr:   nestedLockErr,
			err:             nestedLockErr,
		},
		{
			name:          "both_errors",
			ownLockErr:    ownLockErr,
			nestedLockErr: nestedLockErr,
			err:           ownLockErr,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset from any previous runs
			ownUnlockCalled = false
			ownCalled := false
			nestedCalled := false
			nl := nestedLocker{
				own: FuncMockLocker{
					LockF: func(key string, options ...Option) (RuleLock, error) {
						assert.Equal(t, "key", key)
						ownCalled = true
						return testOwnLock, tc.ownLockErr
					},
				},
				nested: FuncMockLocker{
					LockF: func(key string, options ...Option) (RuleLock, error) {
						// The own locker should have been called first
						assert.True(t, ownCalled)
						assert.Equal(t, "key", key)
						nestedCalled = true
						return testNestedLock, tc.nestedLockErr
					},
				},
			}
			var err error
			lock, err := nl.Lock("key")
			assert.Equal(t, tc.nestedCalled, nestedCalled)
			assert.Equal(t, tc.ownUnlockCalled, ownUnlockCalled)
			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error())
				return
			}
			assert.NoError(t, err)
			nLock, ok := lock.(nestedLock)
			if assert.True(t, ok) {
				getVal := func(rl RuleLock) string {
					tl, ok := rl.(testLock)
					if !ok {
						return ""
					}
					return tl.val
				}
				assert.Equal(t, testOwnLock.val, getVal(nLock.own))
				assert.Equal(t, testNestedLock.val, getVal(nLock.nested))
			}
		})
	}
}
