/*
Copyright 2018 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package buildcontext

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"

	"github.com/Azure/azure-storage-file-go/azfile"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
)

const storageAccountEnvName = "AZURE_STORAGE_ACCOUNT"
const azureStorageAccountKeyEnvName = "AZURE_STORAGE_ACCOUNT_KEY"

// Azure unifies calls to download and unpack the build context.
type Azure struct {
	context string
}

// UnpackTarFromBuildContext download and untar a file from Azure
func (s *Azure) UnpackTarFromBuildContext() (string, error) {
	bucket, item := util.GetBucketAndItem(s.context)
	storageAccount, accountKey := accountInfo()
	credential, err := azfile.NewSharedKeyCredential(storageAccount, accountKey)
	if err != nil {
		return bucket, err
	}
	u, _ := url.Parse(fmt.Sprintf("https://%s.file.core.windows.net/%s/%s", storageAccount, bucket, item))
	fileURL := azfile.NewFileURL(*u, azfile.NewPipeline(credential, azfile.PipelineOptions{}))
	directory := constants.BuildContextDir
	tarPath := filepath.Join(directory, constants.ContextTar)
	if err := os.MkdirAll(directory, 0750); err != nil {
		return directory, err
	}
	file, err := os.Create(tarPath)
	if err != nil {
		return directory, err
	}
	_, err = azfile.DownloadAzureFileToFile(context.Background(), fileURL, file,
		azfile.DownloadFromAzureFileOptions{
			Parallelism:              2,
			MaxRetryRequestsPerRange: 2,
			Progress: func(bytesTransferred int64) {
				fmt.Printf("Downloaded %d bytes.\n", bytesTransferred)
			},
		})
	if err != nil {
		return directory, err
	}
	return directory, util.UnpackCompressedTar(tarPath, directory)
}

func accountInfo() (string, string) {
	return getEnv(storageAccountEnvName), getEnv(azureStorageAccountKeyEnvName)
}

func getEnv(env string) string {
	storageAccount, ok := os.LookupEnv(env)
	if !ok {
		log.Fatal(fmt.Sprintf("Need %s env", env))
	}
	return storageAccount
}
