package v1alpha1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestArtifactPluginName(t *testing.T) {
	pluginName := ArtifactPluginName("my-plugin")

	t.Run("SocketDir", func(t *testing.T) {
		expected := "/tmp/artifact-plugins/my-plugin"
		assert.Equal(t, expected, pluginName.SocketDir())
	})

	t.Run("SocketPath", func(t *testing.T) {
		expected := "/tmp/artifact-plugins/my-plugin/socket"
		assert.Equal(t, expected, pluginName.SocketPath())
	})

	t.Run("VolumeMount", func(t *testing.T) {
		volumeMount := pluginName.VolumeMount()
		expected := apiv1.VolumeMount{
			Name:      "artifact-plugin-my-plugin",
			MountPath: "/tmp/artifact-plugins/my-plugin",
		}
		assert.Equal(t, expected, volumeMount)
	})

	t.Run("Volume", func(t *testing.T) {
		volume := pluginName.Volume()
		expected := apiv1.Volume{
			Name: "artifact-plugin-my-plugin",
			VolumeSource: apiv1.VolumeSource{
				EmptyDir: &apiv1.EmptyDirVolumeSource{},
			},
		}
		assert.Equal(t, expected, volume)
	})

	t.Run("PluginNameWithSpecialChars", func(t *testing.T) {
		specialPlugin := ArtifactPluginName("my-plugin-v1.2.3")
		assert.Equal(t, "/tmp/artifact-plugins/my-plugin-v1.2.3", specialPlugin.SocketDir())
		assert.Equal(t, "artifact-plugin-my-plugin-v1.2.3", specialPlugin.Volume().Name)
	})
}

func TestPluginArtifact(t *testing.T) {
	t.Run("HasLocation_Complete", func(t *testing.T) {
		plugin := &PluginArtifact{
			Name:          "test-plugin",
			Configuration: `{"bucket": "my-bucket"}`,
			Key:           "path/to/artifact",
		}
		assert.True(t, plugin.HasLocation())
	})

	t.Run("HasLocation_MissingName", func(t *testing.T) {
		plugin := &PluginArtifact{
			Name:          "",
			Configuration: `{"bucket": "my-bucket"}`,
			Key:           "path/to/artifact",
		}
		assert.False(t, plugin.HasLocation())
	})

	t.Run("HasLocation_MissingConfiguration", func(t *testing.T) {
		plugin := &PluginArtifact{
			Name:          "test-plugin",
			Configuration: "",
			Key:           "path/to/artifact",
		}
		assert.False(t, plugin.HasLocation())
	})

	t.Run("HasLocation_MissingKey", func(t *testing.T) {
		plugin := &PluginArtifact{
			Name:          "test-plugin",
			Configuration: `{"bucket": "my-bucket"}`,
			Key:           "",
		}
		assert.False(t, plugin.HasLocation())
	})

	t.Run("HasLocation_Nil", func(t *testing.T) {
		var plugin *PluginArtifact
		assert.False(t, plugin.HasLocation())
	})

	t.Run("ConnectionTimeoutSeconds", func(t *testing.T) {
		plugin := &PluginArtifact{
			Name:                     "test-plugin",
			Configuration:            `{"bucket": "my-bucket"}`,
			Key:                      "path/to/artifact",
			ConnectionTimeoutSeconds: 30,
		}
		assert.Equal(t, int32(30), plugin.ConnectionTimeoutSeconds)
		assert.True(t, plugin.HasLocation())
	})
}

func TestPluginArtifactRepository(t *testing.T) {
	t.Run("IntoArtifactLocation_WithKeyFormat", func(t *testing.T) {
		repo := &PluginArtifactRepository{
			Name:          "my-plugin",
			KeyFormat:     "custom/{{workflow.name}}/{{pod.name}}/{{artifact.name}}",
			Configuration: `{"endpoint": "https://my-storage.com"}`,
		}

		location := &ArtifactLocation{}
		repo.IntoArtifactLocation(location)

		require.NotNil(t, location.Plugin)
		assert.Equal(t, ArtifactPluginName("my-plugin"), location.Plugin.Name)
		assert.JSONEq(t, `{"endpoint": "https://my-storage.com"}`, location.Plugin.Configuration)
		assert.Equal(t, "custom/{{workflow.name}}/{{pod.name}}/{{artifact.name}}", location.Plugin.Key)
	})

	t.Run("IntoArtifactLocation_WithoutKeyFormat", func(t *testing.T) {
		repo := &PluginArtifactRepository{
			Name:          "my-plugin",
			Configuration: `{"endpoint": "https://my-storage.com"}`,
		}

		location := &ArtifactLocation{}
		repo.IntoArtifactLocation(location)

		require.NotNil(t, location.Plugin)
		assert.Equal(t, ArtifactPluginName("my-plugin"), location.Plugin.Name)
		assert.JSONEq(t, `{"endpoint": "https://my-storage.com"}`, location.Plugin.Configuration)
		assert.Equal(t, DefaultArchivePattern, location.Plugin.Key)
	})

	t.Run("IntoArtifactLocation_EmptyKeyFormat", func(t *testing.T) {
		repo := &PluginArtifactRepository{
			Name:          "my-plugin",
			KeyFormat:     "",
			Configuration: `{"endpoint": "https://my-storage.com"}`,
		}

		location := &ArtifactLocation{}
		repo.IntoArtifactLocation(location)

		require.NotNil(t, location.Plugin)
		assert.Equal(t, DefaultArchivePattern, location.Plugin.Key)
	})
}

func TestArtifactLocation_Plugin(t *testing.T) {
	t.Run("HasLocation_Plugin", func(t *testing.T) {
		location := &ArtifactLocation{
			Plugin: &PluginArtifact{
				Name:          "test-plugin",
				Configuration: `{"bucket": "my-bucket"}`,
				Key:           "path/to/artifact",
			},
		}
		assert.True(t, location.HasLocation())
	})

	t.Run("HasLocation_PluginIncomplete", func(t *testing.T) {
		location := &ArtifactLocation{
			Plugin: &PluginArtifact{
				Name:          "test-plugin",
				Configuration: "",
				Key:           "path/to/artifact",
			},
		}
		assert.False(t, location.HasLocation())
	})
}

func TestArtifacts_GetPluginNames(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.NewTestLogger(logging.Info, logging.JSON))

	t.Run("NoPlugins", func(t *testing.T) {
		artifacts := Artifacts{
			{
				Name: "regular-artifact",
				ArtifactLocation: ArtifactLocation{
					S3: &S3Artifact{
						S3Bucket: S3Bucket{Bucket: "my-bucket"},
						Key:      "path/to/artifact",
					},
				},
			},
		}

		pluginNames := artifacts.GetPluginNames(ctx, nil, ExcludeLogs, nil)
		assert.Empty(t, pluginNames)
	})

	t.Run("SinglePlugin", func(t *testing.T) {
		artifacts := Artifacts{
			{
				Name: "plugin-artifact",
				ArtifactLocation: ArtifactLocation{
					Plugin: &PluginArtifact{
						Name:          "my-plugin",
						Configuration: `{"bucket": "my-bucket"}`,
						Key:           "path/to/artifact",
					},
				},
			},
		}

		pluginNames := artifacts.GetPluginNames(ctx, nil, ExcludeLogs, nil)
		assert.Len(t, pluginNames, 1)
		assert.Contains(t, pluginNames, ArtifactPluginName("my-plugin"))
	})

	t.Run("MultiplePlugins", func(t *testing.T) {
		artifacts := Artifacts{
			{
				Name: "plugin-artifact-1",
				ArtifactLocation: ArtifactLocation{
					Plugin: &PluginArtifact{
						Name:          "plugin-1",
						Configuration: `{"bucket": "bucket-1"}`,
						Key:           "path/to/artifact1",
					},
				},
			},
			{
				Name: "plugin-artifact-2",
				ArtifactLocation: ArtifactLocation{
					Plugin: &PluginArtifact{
						Name:          "plugin-2",
						Configuration: `{"bucket": "bucket-2"}`,
						Key:           "path/to/artifact2",
					},
				},
			},
		}

		pluginNames := artifacts.GetPluginNames(ctx, nil, ExcludeLogs, nil)
		assert.Len(t, pluginNames, 2)
		assert.Contains(t, pluginNames, ArtifactPluginName("plugin-1"))
		assert.Contains(t, pluginNames, ArtifactPluginName("plugin-2"))
	})

	t.Run("DuplicatePlugins", func(t *testing.T) {
		artifacts := Artifacts{
			{
				Name: "plugin-artifact-1",
				ArtifactLocation: ArtifactLocation{
					Plugin: &PluginArtifact{
						Name:          "my-plugin",
						Configuration: `{"bucket": "bucket-1"}`,
						Key:           "path/to/artifact1",
					},
				},
			},
			{
				Name: "plugin-artifact-2",
				ArtifactLocation: ArtifactLocation{
					Plugin: &PluginArtifact{
						Name:          "my-plugin",
						Configuration: `{"bucket": "bucket-2"}`,
						Key:           "path/to/artifact2",
					},
				},
			},
		}

		pluginNames := artifacts.GetPluginNames(ctx, nil, ExcludeLogs, nil)
		assert.Len(t, pluginNames, 1)
		assert.Contains(t, pluginNames, ArtifactPluginName("my-plugin"))
	})

	t.Run("WithDefaultRepo", func(t *testing.T) {
		artifacts := Artifacts{
			{
				Name:             "artifact-without-plugin",
				ArtifactLocation: ArtifactLocation{
					// No plugin specified, should use default repo
				},
			},
		}

		defaultRepo := &ArtifactRepository{
			Plugin: &PluginArtifactRepository{
				Name:          "default-plugin",
				Configuration: `{"bucket": "default-bucket"}`,
			},
		}

		pluginNames := artifacts.GetPluginNames(ctx, defaultRepo, ExcludeLogs, nil)
		assert.Len(t, pluginNames, 1)
		assert.Contains(t, pluginNames, ArtifactPluginName("default-plugin"))
	})

	t.Run("IncludeLogs", func(t *testing.T) {
		artifacts := Artifacts{
			{
				Name: "regular-artifact",
				ArtifactLocation: ArtifactLocation{
					S3: &S3Artifact{
						S3Bucket: S3Bucket{Bucket: "my-bucket"},
						Key:      "path/to/artifact",
					},
				},
			},
		}

		defaultRepo := &ArtifactRepository{
			Plugin: &PluginArtifactRepository{
				Name:          "log-plugin",
				Configuration: `{"bucket": "log-bucket"}`,
			},
			ArchiveLogs: new(true),
		}

		pluginNames := artifacts.GetPluginNames(ctx, defaultRepo, IncludeLogs, nil)
		assert.Len(t, pluginNames, 1)
		assert.Contains(t, pluginNames, ArtifactPluginName("log-plugin"))
	})

	t.Run("ExcludeLogs", func(t *testing.T) {
		artifacts := Artifacts{
			{
				Name: "regular-artifact",
				ArtifactLocation: ArtifactLocation{
					S3: &S3Artifact{
						S3Bucket: S3Bucket{Bucket: "my-bucket"},
						Key:      "path/to/artifact",
					},
				},
			},
		}

		defaultRepo := &ArtifactRepository{
			S3: &S3ArtifactRepository{
				S3Bucket: S3Bucket{Bucket: "log-bucket"},
			},
			ArchiveLogs: new(true),
		}

		pluginNames := artifacts.GetPluginNames(ctx, defaultRepo, ExcludeLogs, nil)
		// When ExcludeLogs is used and there are no plugin artifacts, should be empty
		assert.Empty(t, pluginNames)
	})

	t.Run("MixedArtifacts", func(t *testing.T) {
		artifacts := Artifacts{
			{
				Name: "s3-artifact",
				ArtifactLocation: ArtifactLocation{
					S3: &S3Artifact{
						S3Bucket: S3Bucket{Bucket: "s3-bucket"},
						Key:      "path/to/s3-artifact",
					},
				},
			},
			{
				Name: "plugin-artifact",
				ArtifactLocation: ArtifactLocation{
					Plugin: &PluginArtifact{
						Name:          "my-plugin",
						Configuration: `{"bucket": "plugin-bucket"}`,
						Key:           "path/to/plugin-artifact",
					},
				},
			},
			{
				Name:             "default-artifact",
				ArtifactLocation: ArtifactLocation{
					// No specific location, should use default
				},
			},
		}

		defaultRepo := &ArtifactRepository{
			Plugin: &PluginArtifactRepository{
				Name:          "default-plugin",
				Configuration: `{"bucket": "default-bucket"}`,
			},
		}

		pluginNames := artifacts.GetPluginNames(ctx, defaultRepo, ExcludeLogs, nil)
		assert.Len(t, pluginNames, 2)
		assert.Contains(t, pluginNames, ArtifactPluginName("my-plugin"))
		assert.Contains(t, pluginNames, ArtifactPluginName("default-plugin"))
	})

	t.Run("MultiplePluginsWithDefaultRepo", func(t *testing.T) {
		artifacts := Artifacts{
			{
				Name: "plugin-artifact-1",
				ArtifactLocation: ArtifactLocation{
					Plugin: &PluginArtifact{
						Name:          "plugin-1",
						Configuration: `{"bucket": "bucket-1"}`,
						Key:           "path/to/artifact1",
					},
				},
			},
			{
				Name: "plugin-artifact-2",
				ArtifactLocation: ArtifactLocation{
					Plugin: &PluginArtifact{
						Name:          "plugin-2",
						Configuration: `{"bucket": "bucket-2"}`,
						Key:           "path/to/artifact2",
					},
				},
			},
			{
				Name:             "default-artifact",
				ArtifactLocation: ArtifactLocation{
					// No specific location, should use default repo
				},
			},
		}

		defaultRepo := &ArtifactRepository{
			Plugin: &PluginArtifactRepository{
				Name:          "default-plugin",
				Configuration: `{"bucket": "default-bucket"}`,
			},
			ArchiveLogs: new(true),
		}

		pluginNames := artifacts.GetPluginNames(ctx, defaultRepo, IncludeLogs, nil)
		assert.Len(t, pluginNames, 3)
		assert.Contains(t, pluginNames, ArtifactPluginName("plugin-1"))
		assert.Contains(t, pluginNames, ArtifactPluginName("plugin-2"))
		assert.Contains(t, pluginNames, ArtifactPluginName("default-plugin"))
	})

	t.Run("MultiplePluginsWithLogging", func(t *testing.T) {
		artifacts := Artifacts{
			{
				Name: "s3-artifact",
				ArtifactLocation: ArtifactLocation{
					S3: &S3Artifact{
						S3Bucket: S3Bucket{Bucket: "s3-bucket"},
						Key:      "path/to/s3-artifact",
					},
				},
			},
			{
				Name: "plugin-artifact-1",
				ArtifactLocation: ArtifactLocation{
					Plugin: &PluginArtifact{
						Name:          "storage-plugin",
						Configuration: `{"endpoint": "https://storage1.com"}`,
						Key:           "path/to/plugin-artifact1",
					},
				},
			},
			{
				Name: "plugin-artifact-2",
				ArtifactLocation: ArtifactLocation{
					Plugin: &PluginArtifact{
						Name:          "backup-plugin",
						Configuration: `{"endpoint": "https://backup.com"}`,
						Key:           "path/to/plugin-artifact2",
					},
				},
			},
		}

		defaultRepo := &ArtifactRepository{
			Plugin: &PluginArtifactRepository{
				Name:          "log-plugin",
				Configuration: `{"endpoint": "https://logs.com"}`,
			},
			ArchiveLogs: new(true),
		}

		pluginNames := artifacts.GetPluginNames(ctx, defaultRepo, IncludeLogs, nil)
		assert.Len(t, pluginNames, 3)
		assert.Contains(t, pluginNames, ArtifactPluginName("storage-plugin"))
		assert.Contains(t, pluginNames, ArtifactPluginName("backup-plugin"))
		assert.Contains(t, pluginNames, ArtifactPluginName("log-plugin"))
	})

	t.Run("SamePluginMultipleConfigurations", func(t *testing.T) {
		artifacts := Artifacts{
			{
				Name: "plugin-artifact-1",
				ArtifactLocation: ArtifactLocation{
					Plugin: &PluginArtifact{
						Name:          "my-plugin",
						Configuration: `{"bucket": "bucket-1", "region": "us-east-1"}`,
						Key:           "path/to/artifact1",
					},
				},
			},
			{
				Name: "plugin-artifact-2",
				ArtifactLocation: ArtifactLocation{
					Plugin: &PluginArtifact{
						Name:          "my-plugin",
						Configuration: `{"bucket": "bucket-2", "region": "us-west-2"}`,
						Key:           "path/to/artifact2",
					},
				},
			},
			{
				Name: "plugin-artifact-3",
				ArtifactLocation: ArtifactLocation{
					Plugin: &PluginArtifact{
						Name:          "other-plugin",
						Configuration: `{"endpoint": "https://other.com"}`,
						Key:           "path/to/artifact3",
					},
				},
			},
		}

		pluginNames := artifacts.GetPluginNames(ctx, nil, ExcludeLogs, nil)
		// Should only have 2 unique plugin names despite 3 artifacts
		assert.Len(t, pluginNames, 2)
		assert.Contains(t, pluginNames, ArtifactPluginName("my-plugin"))
		assert.Contains(t, pluginNames, ArtifactPluginName("other-plugin"))
	})

	t.Run("ComplexMultiPluginScenario", func(t *testing.T) {
		artifacts := Artifacts{
			{
				Name: "input-artifact",
				ArtifactLocation: ArtifactLocation{
					Plugin: &PluginArtifact{
						Name:          "input-plugin",
						Configuration: `{"source": "external"}`,
						Key:           "inputs/data.json",
					},
				},
			},
			{
				Name: "processing-artifact",
				ArtifactLocation: ArtifactLocation{
					Plugin: &PluginArtifact{
						Name:          "processing-plugin",
						Configuration: `{"temp": true}`,
						Key:           "temp/processing.dat",
					},
				},
			},
			{
				Name: "output-artifact",
				ArtifactLocation: ArtifactLocation{
					Plugin: &PluginArtifact{
						Name:          "output-plugin",
						Configuration: `{"destination": "final"}`,
						Key:           "outputs/result.json",
					},
				},
			},
			{
				Name: "backup-artifact",
				ArtifactLocation: ArtifactLocation{
					Plugin: &PluginArtifact{
						Name:          "backup-plugin",
						Configuration: `{"retention": "30d"}`,
						Key:           "backups/result-backup.json",
					},
				},
			},
			{
				Name: "s3-artifact",
				ArtifactLocation: ArtifactLocation{
					S3: &S3Artifact{
						S3Bucket: S3Bucket{Bucket: "legacy-bucket"},
						Key:      "legacy/data.json",
					},
				},
			},
			{
				Name:             "default-artifact",
				ArtifactLocation: ArtifactLocation{
					// Uses default repo
				},
			},
		}

		defaultRepo := &ArtifactRepository{
			Plugin: &PluginArtifactRepository{
				Name:          "default-plugin",
				Configuration: `{"default": true}`,
			},
			ArchiveLogs: new(true),
		}

		pluginNames := artifacts.GetPluginNames(ctx, defaultRepo, IncludeLogs, nil)
		// Should have 5 unique plugins: input, processing, output, backup, default (for both default artifact and logs)
		assert.Len(t, pluginNames, 5)
		assert.Contains(t, pluginNames, ArtifactPluginName("input-plugin"))
		assert.Contains(t, pluginNames, ArtifactPluginName("processing-plugin"))
		assert.Contains(t, pluginNames, ArtifactPluginName("output-plugin"))
		assert.Contains(t, pluginNames, ArtifactPluginName("backup-plugin"))
		assert.Contains(t, pluginNames, ArtifactPluginName("default-plugin"))
	})

	t.Run("ArchiveLocation", func(t *testing.T) {
		t.Run("WithPlugin", func(t *testing.T) {
			// Test that archiveLocation plugin is used when artifact doesn't have explicit plugin
			artifacts := Artifacts{
				{
					Name:             "artifact-without-plugin",
					ArtifactLocation: ArtifactLocation{
						// No plugin specified, should use archiveLocation
					},
				},
			}

			archiveLocation := &ArtifactLocation{
				Plugin: &PluginArtifact{
					Name:          "archive-plugin",
					Configuration: `{"bucket": "archive-bucket"}`,
				},
			}

			pluginNames := artifacts.GetPluginNames(ctx, nil, ExcludeLogs, archiveLocation)
			assert.Len(t, pluginNames, 1)
			assert.Contains(t, pluginNames, ArtifactPluginName("archive-plugin"))
		})

		t.Run("PluginNotUsedWhenArtifactHasPlugin", func(t *testing.T) {
			// Test that artifact's explicit plugin takes priority over archiveLocation
			artifacts := Artifacts{
				{
					Name: "artifact-with-plugin",
					ArtifactLocation: ArtifactLocation{
						Plugin: &PluginArtifact{
							Name:          "artifact-plugin",
							Configuration: `{"bucket": "artifact-bucket"}`,
							Key:           "path/to/artifact",
						},
					},
				},
			}

			archiveLocation := &ArtifactLocation{
				Plugin: &PluginArtifact{
					Name:          "archive-plugin",
					Configuration: `{"bucket": "archive-bucket"}`,
				},
			}

			pluginNames := artifacts.GetPluginNames(ctx, nil, ExcludeLogs, archiveLocation)
			assert.Len(t, pluginNames, 1)
			assert.Contains(t, pluginNames, ArtifactPluginName("artifact-plugin"))
			assert.NotContains(t, pluginNames, ArtifactPluginName("archive-plugin"))
		})

		t.Run("PriorityOverDefaultRepo", func(t *testing.T) {
			// Test that archiveLocation takes priority over defaultRepo
			artifacts := Artifacts{
				{
					Name:             "artifact-without-plugin",
					ArtifactLocation: ArtifactLocation{
						// No plugin specified
					},
				},
			}

			defaultRepo := &ArtifactRepository{
				Plugin: &PluginArtifactRepository{
					Name:          "default-plugin",
					Configuration: `{"bucket": "default-bucket"}`,
				},
			}

			archiveLocation := &ArtifactLocation{
				Plugin: &PluginArtifact{
					Name:          "archive-plugin",
					Configuration: `{"bucket": "archive-bucket"}`,
				},
			}

			pluginNames := artifacts.GetPluginNames(ctx, defaultRepo, ExcludeLogs, archiveLocation)
			assert.Len(t, pluginNames, 1)
			assert.Contains(t, pluginNames, ArtifactPluginName("archive-plugin"))
			assert.NotContains(t, pluginNames, ArtifactPluginName("default-plugin"))
		})

		t.Run("WithMultipleArtifacts", func(t *testing.T) {
			// Test archiveLocation with mix of artifacts with and without explicit plugins
			artifacts := Artifacts{
				{
					Name: "artifact-with-plugin",
					ArtifactLocation: ArtifactLocation{
						Plugin: &PluginArtifact{
							Name:          "specific-plugin",
							Configuration: `{"bucket": "specific-bucket"}`,
							Key:           "path/to/artifact1",
						},
					},
				},
				{
					Name:             "artifact-without-plugin-1",
					ArtifactLocation: ArtifactLocation{
						// Should use archiveLocation
					},
				},
				{
					Name:             "artifact-without-plugin-2",
					ArtifactLocation: ArtifactLocation{
						// Should use archiveLocation
					},
				},
			}

			archiveLocation := &ArtifactLocation{
				Plugin: &PluginArtifact{
					Name:          "archive-plugin",
					Configuration: `{"bucket": "archive-bucket"}`,
				},
			}

			pluginNames := artifacts.GetPluginNames(ctx, nil, ExcludeLogs, archiveLocation)
			assert.Len(t, pluginNames, 2)
			assert.Contains(t, pluginNames, ArtifactPluginName("specific-plugin"))
			assert.Contains(t, pluginNames, ArtifactPluginName("archive-plugin"))
		})

		t.Run("NilDoesNotCrash", func(t *testing.T) {
			// Test that nil archiveLocation doesn't cause issues
			artifacts := Artifacts{
				{
					Name:             "artifact-without-plugin",
					ArtifactLocation: ArtifactLocation{},
				},
			}

			defaultRepo := &ArtifactRepository{
				Plugin: &PluginArtifactRepository{
					Name:          "default-plugin",
					Configuration: `{"bucket": "default-bucket"}`,
				},
			}

			pluginNames := artifacts.GetPluginNames(ctx, defaultRepo, ExcludeLogs, nil)
			assert.Len(t, pluginNames, 1)
			assert.Contains(t, pluginNames, ArtifactPluginName("default-plugin"))
		})

		t.Run("WithS3FallsBackToDefault", func(t *testing.T) {
			// Test that archiveLocation with S3 (not plugin) falls back to defaultRepo for plugin
			artifacts := Artifacts{
				{
					Name:             "artifact-without-plugin",
					ArtifactLocation: ArtifactLocation{},
				},
			}

			defaultRepo := &ArtifactRepository{
				Plugin: &PluginArtifactRepository{
					Name:          "default-plugin",
					Configuration: `{"bucket": "default-bucket"}`,
				},
			}

			archiveLocation := &ArtifactLocation{
				S3: &S3Artifact{
					S3Bucket: S3Bucket{Bucket: "s3-bucket"},
				},
			}

			pluginNames := artifacts.GetPluginNames(ctx, defaultRepo, ExcludeLogs, archiveLocation)
			assert.Len(t, pluginNames, 1)
			assert.Contains(t, pluginNames, ArtifactPluginName("default-plugin"))
		})

		t.Run("WithPluginAndLogging", func(t *testing.T) {
			// Test archiveLocation plugin with logging enabled
			artifacts := Artifacts{
				{
					Name:             "artifact-without-plugin",
					ArtifactLocation: ArtifactLocation{},
				},
			}

			defaultRepo := &ArtifactRepository{
				Plugin: &PluginArtifactRepository{
					Name:          "log-plugin",
					Configuration: `{"bucket": "log-bucket"}`,
				},
				ArchiveLogs: new(true),
			}

			archiveLocation := &ArtifactLocation{
				Plugin: &PluginArtifact{
					Name:          "archive-plugin",
					Configuration: `{"bucket": "archive-bucket"}`,
				},
			}

			pluginNames := artifacts.GetPluginNames(ctx, defaultRepo, IncludeLogs, archiveLocation)
			// Should have both: archive-plugin for artifacts and log-plugin for logging
			assert.Len(t, pluginNames, 2)
			assert.Contains(t, pluginNames, ArtifactPluginName("archive-plugin"))
			assert.Contains(t, pluginNames, ArtifactPluginName("log-plugin"))
		})

		t.Run("EmptyPluginFallsBackToDefault", func(t *testing.T) {
			// Test that archiveLocation with empty plugin configuration falls back to defaultRepo
			artifacts := Artifacts{
				{
					Name:             "artifact-without-plugin",
					ArtifactLocation: ArtifactLocation{},
				},
			}

			defaultRepo := &ArtifactRepository{
				Plugin: &PluginArtifactRepository{
					Name:          "default-plugin",
					Configuration: `{"bucket": "default-bucket"}`,
				},
			}

			archiveLocation := &ArtifactLocation{
				Plugin: &PluginArtifact{
					// Empty name should not be used
					Name:          "",
					Configuration: "",
				},
			}

			pluginNames := artifacts.GetPluginNames(ctx, defaultRepo, ExcludeLogs, archiveLocation)
			assert.Len(t, pluginNames, 1)
			assert.Contains(t, pluginNames, ArtifactPluginName("default-plugin"))
		})

		t.Run("WithDifferentPluginsThanArtifacts", func(t *testing.T) {
			// Test complex scenario with multiple different plugin sources
			artifacts := Artifacts{
				{
					Name: "explicit-plugin-artifact",
					ArtifactLocation: ArtifactLocation{
						Plugin: &PluginArtifact{
							Name:          "explicit-plugin",
							Configuration: `{"bucket": "explicit-bucket"}`,
							Key:           "path/to/artifact",
						},
					},
				},
				{
					Name:             "archive-location-artifact",
					ArtifactLocation: ArtifactLocation{
						// Should use archiveLocation
					},
				},
			}

			defaultRepo := &ArtifactRepository{
				Plugin: &PluginArtifactRepository{
					Name:          "default-plugin",
					Configuration: `{"bucket": "default-bucket"}`,
				},
			}

			archiveLocation := &ArtifactLocation{
				Plugin: &PluginArtifact{
					Name:          "archive-plugin",
					Configuration: `{"bucket": "archive-bucket"}`,
				},
			}

			pluginNames := artifacts.GetPluginNames(ctx, defaultRepo, ExcludeLogs, archiveLocation)
			assert.Len(t, pluginNames, 2)
			assert.Contains(t, pluginNames, ArtifactPluginName("explicit-plugin"))
			assert.Contains(t, pluginNames, ArtifactPluginName("archive-plugin"))
			// Default plugin should not be included since archiveLocation has priority
			assert.NotContains(t, pluginNames, ArtifactPluginName("default-plugin"))
		})
	})
}

func TestMultiplePluginArtifactRepositories(t *testing.T) {
	t.Run("DifferentPluginRepositories", func(t *testing.T) {
		repo1 := &PluginArtifactRepository{
			Name:          "plugin-1",
			KeyFormat:     "repo1/{{workflow.name}}/{{pod.name}}",
			Configuration: `{"endpoint": "https://repo1.com"}`,
		}

		repo2 := &PluginArtifactRepository{
			Name:          "plugin-2",
			KeyFormat:     "repo2/{{workflow.name}}/{{pod.name}}",
			Configuration: `{"endpoint": "https://repo2.com"}`,
		}

		location1 := &ArtifactLocation{}
		repo1.IntoArtifactLocation(location1)

		location2 := &ArtifactLocation{}
		repo2.IntoArtifactLocation(location2)

		// Verify both locations are configured correctly
		require.NotNil(t, location1.Plugin)
		require.NotNil(t, location2.Plugin)

		assert.Equal(t, ArtifactPluginName("plugin-1"), location1.Plugin.Name)
		assert.Equal(t, "repo1/{{workflow.name}}/{{pod.name}}", location1.Plugin.Key)
		assert.JSONEq(t, `{"endpoint": "https://repo1.com"}`, location1.Plugin.Configuration)

		assert.Equal(t, ArtifactPluginName("plugin-2"), location2.Plugin.Name)
		assert.Equal(t, "repo2/{{workflow.name}}/{{pod.name}}", location2.Plugin.Key)
		assert.JSONEq(t, `{"endpoint": "https://repo2.com"}`, location2.Plugin.Configuration)
	})

	t.Run("SamePluginDifferentConfigurations", func(t *testing.T) {
		repo1 := &PluginArtifactRepository{
			Name:          "shared-plugin",
			KeyFormat:     "env1/{{workflow.name}}/{{pod.name}}",
			Configuration: `{"environment": "production", "region": "us-east-1"}`,
		}

		repo2 := &PluginArtifactRepository{
			Name:          "shared-plugin",
			KeyFormat:     "env2/{{workflow.name}}/{{pod.name}}",
			Configuration: `{"environment": "staging", "region": "us-west-2"}`,
		}

		location1 := &ArtifactLocation{}
		repo1.IntoArtifactLocation(location1)

		location2 := &ArtifactLocation{}
		repo2.IntoArtifactLocation(location2)

		// Both should use the same plugin name but different configurations
		require.NotNil(t, location1.Plugin)
		require.NotNil(t, location2.Plugin)

		assert.Equal(t, ArtifactPluginName("shared-plugin"), location1.Plugin.Name)
		assert.Equal(t, ArtifactPluginName("shared-plugin"), location2.Plugin.Name)

		assert.Equal(t, "env1/{{workflow.name}}/{{pod.name}}", location1.Plugin.Key)
		assert.Equal(t, "env2/{{workflow.name}}/{{pod.name}}", location2.Plugin.Key)

		assert.JSONEq(t, `{"environment": "production", "region": "us-east-1"}`, location1.Plugin.Configuration)
		assert.JSONEq(t, `{"environment": "staging", "region": "us-west-2"}`, location2.Plugin.Configuration)
	})
}
