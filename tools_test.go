package toolkit

import (
    "testing"
)

const randomStringLength = 10
const expectedRandomStringLength = 10

func TestTools_RandomString(t *testing.T) {
    var testTools Tools
    s := testTools.RandomString(randomStringLength)
    if len(s) != expectedRandomStringLength {
        t.Error("wrong length random string returned...")
    } else {
        t.Log("RandomString.length is equal to ", randomStringLength)
    }
}