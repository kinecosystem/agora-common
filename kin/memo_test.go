package kin

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMemo_Valid(t *testing.T) {
	var empyFK [29]byte

	for v := byte(0); v <= 7; v++ {
		m, err := NewMemo(v, TransactionTypeEarn, 1, make([]byte, 29))
		require.NoError(t, err)

		require.EqualValues(t, magicByte, m[0]&0x3)
		require.EqualValues(t, v, m.Version())
		require.EqualValues(t, TransactionTypeEarn, m.TransactionType())
		require.EqualValues(t, 1, m.AppIndex())
		require.EqualValues(t, empyFK, m.ForeignKey())
	}

	for txType := TransactionTypeUnknown; txType <= MaxTransactionType; txType++ {
		m, err := NewMemo(1, txType, 1, make([]byte, 29))
		require.NoError(t, err)

		require.EqualValues(t, magicByte, m[0]&0x3)
		require.EqualValues(t, 1, m.Version())
		require.EqualValues(t, txType, m.TransactionType())
		require.EqualValues(t, 1, m.AppIndex())
		require.EqualValues(t, empyFK, m.ForeignKey())
	}

	for i := uint16(0); i < math.MaxUint16; i++ {
		m, err := NewMemo(1, TransactionTypeEarn, i, make([]byte, 29))
		require.NoError(t, err)

		require.EqualValues(t, magicByte, m[0]&0x3)
		require.EqualValues(t, 1, m.Version())
		require.EqualValues(t, TransactionTypeEarn, m.TransactionType())
		require.EqualValues(t, i, m.AppIndex())
		require.EqualValues(t, empyFK, m.ForeignKey())
	}

	fk := make([]byte, 29)
	for i := byte(0); i < 28; i++ {
		fk[i] = i
	}
	fk[28] = 0xff

	m, err := NewMemo(1, TransactionTypeEarn, 2, fk)
	require.NoError(t, err)

	actual := m.ForeignKey()
	for i := byte(0); i < 28; i++ {
		require.Equal(t, fk[i], actual[i])
	}

	// Note, because we only have 230 bits, the last fk byte
	// only technically has 6 bits. As a result, if we have 0xff
	// in the last byte, we should only see 0x3f, which is 0xff >> 2.
	require.Equal(t, byte(0x3f), actual[28])
}

func TestMemo_Invalid(t *testing.T) {
	_, err := NewMemo(8, TransactionTypeEarn, 1, make([]byte, 29))
	require.NotNil(t, err)

	_, err = NewMemo(8, TransactionTypeP2P+1, 1, make([]byte, 29))
	require.NotNil(t, err)

	m, err := NewMemo(1, TransactionTypeEarn, 1, make([]byte, 29))
	require.Nil(t, err)
	require.True(t, IsValidMemo(m))
	require.True(t, IsValidMemoStrict(m))

	// Invalid magic byte
	m[0] &= 0xfc
	require.False(t, IsValidMemo(m))
	require.False(t, IsValidMemoStrict(m))

	// Invalid transaction type
	m, err = NewMemo(1, TransactionTypeUnknown, 1, make([]byte, 29))
	require.Nil(t, err)
	require.False(t, IsValidMemo(m))
	require.False(t, IsValidMemoStrict(m))

	// Version higher than configured
	m, err = NewMemo(7, TransactionTypeEarn, 1, make([]byte, 29))
	require.Nil(t, err)
	require.True(t, IsValidMemo(m))
	require.False(t, IsValidMemoStrict(m))

	// Transaction type higher than configured
	m, err = NewMemo(1, MaxTransactionType+1, 1, make([]byte, 29))
	require.Nil(t, err)
	require.True(t, IsValidMemo(m))
	require.False(t, IsValidMemoStrict(m))
}
