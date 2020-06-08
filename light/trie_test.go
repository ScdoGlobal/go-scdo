/**
*  @file
*  @copyright defined in slc/LICENSE
 */

package light

import (
	"testing"

	"github.com/seeledevteam/slc/common"
	"github.com/seeledevteam/slc/database/leveldb"
	"github.com/seeledevteam/slc/trie"
	"github.com/stretchr/testify/assert"
)

type mockOdrRetriever struct {
	resp odrResponse
}

func (r *mockOdrRetriever) retrieveWithFilter(request odrRequest, filter peerFilter) (odrResponse, error) {
	return r.resp, nil
}

func Test_Trie_Get(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	// prepare trie on server side
	dbPrefix := []byte("test prefix")
	trie := trie.NewEmptyTrie(dbPrefix, db)
	trie.Put([]byte("hello"), []byte("HELLO"))
	trie.Put([]byte("seeleCredo"), []byte("SEELECREDO"))
	trie.Put([]byte("world"), []byte("WORLD"))

	// prepare mock odr retriever
	proof, err := trie.GetProof([]byte("seeleCredo"))
	assert.Nil(t, err)
	retriever := &mockOdrRetriever{
		resp: &odrTriePoof{
			Proof: mapToArray(proof),
		},
	}

	// validate on light client
	lightTrie := newOdrTrie(retriever, trie.Hash(), dbPrefix, common.EmptyHash)

	// key exists
	v, ok, err := lightTrie.Get([]byte("seeleCredo"))
	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Equal(t, []byte("SEELECREDO"), v)

	// key not found
	v, ok, err = lightTrie.Get([]byte("seeleCredo 2"))
	assert.Nil(t, err)
	assert.False(t, ok)
	assert.Nil(t, v)
}
