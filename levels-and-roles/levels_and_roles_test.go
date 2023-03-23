// SPDX-License-Identifier: ice License 1.0

package levelsandroles

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ice-blockchain/eskimo/users"
)

func TestAreLevelsCompleted(t *testing.T) { //nolint:funlen // It's a test function
	t.Parallel()

	testCases := []struct {
		name           string
		actual         *users.Enum[LevelType]
		expectedSubset []LevelType
		expected       bool
	}{
		{
			name:     "expected levels are empty, but actual aren't",
			actual:   &users.Enum[LevelType]{Level1Type},
			expected: false,
		},
		{
			name:     "expected levels are empty and actual as well",
			expected: true,
		},
		{
			name:           "expected levels aren't empty, but actual are empty",
			expectedSubset: []LevelType{Level1Type},
			expected:       false,
		},
		{
			name:           "actual levels are all completed",
			actual:         &users.Enum[LevelType]{Level1Type, Level2Type, Level3Type},
			expectedSubset: []LevelType{Level1Type, Level2Type, Level3Type},
			expected:       true,
		},
		{
			name:           "actual levels are completed, except one",
			actual:         &users.Enum[LevelType]{Level1Type, Level2Type, Level3Type},
			expectedSubset: []LevelType{Level1Type, Level2Type},
			expected:       true,
		},
		{
			name:           "expected levels are not in actual",
			actual:         &users.Enum[LevelType]{Level1Type, Level2Type},
			expectedSubset: []LevelType{Level1Type, Level2Type, Level3Type},
			expected:       false,
		},
	}

	for _, tt := range testCases { //nolint:gocritic // it's a test, no need for optimization
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := AreLevelsCompleted(tt.actual, tt.expectedSubset...)
			require.EqualValues(t, tt.expected, got)
		})
	}
}
