package commands

// func TestDefaultSecureMode(t *testing.T) {
// 	// No certs: We should run insecure
// 	cmd := NewServerCommand()
// 	assert.Equal(t, "false", cmd.Flag("secure").Value.String())

// 	// Clean up and delete tests files
// 	defer func() {
// 		_ = os.Remove("argo-server.crt")
// 		_ = os.Remove("argo-server.key")
// 	}()

// 	_, _ = os.Create("argo-server.crt")
// 	_, _ = os.Create("argo-server.key")

// 	// No certs: We should secure
// 	cmd = NewServerCommand()
// 	assert.Equal(t, "true", cmd.Flag("secure").Value.String())
// }
