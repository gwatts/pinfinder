// +build nodecrypt

package main

var (
	decryptEnabled = false
)

func decrypt(backupDir string, b *backup) {
	b.Status = msgEncryptionDisabled
}
