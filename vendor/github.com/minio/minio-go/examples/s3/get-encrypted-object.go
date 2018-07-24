// +build ignore

/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage (C) 2015 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"io"
	"log"
	"os"

	"github.com/minio/minio-go"
	"github.com/minio/minio-go/pkg/encrypt"
)

func main() {
	// Note: YOUR-ACCESSKEYID, YOUR-SECRETACCESSKEY, my-bucketname, my-objectname and
	// my-testfile are dummy values, please replace them with original values.

	// Requests are always secure (HTTPS) by default. Set secure=false to enable insecure (HTTP) access.
	// This boolean value is the last argument for New().

	// New returns an Amazon S3 compatible client object. API compatibility (v2 or v4) is automatically
	// determined based on the Endpoint value.
	s3Client, err := minio.New("s3.amazonaws.com", "YOUR-ACCESS-KEY-HERE", "YOUR-SECRET-KEY-HERE", true)
	if err != nil {
		log.Fatalln(err)
	}

	//// Build an asymmetric key from private and public files
	//
	// privateKey, err := ioutil.ReadFile("private.key")
	// if err != nil {
	//	t.Fatal(err)
	// }
	//
	// publicKey, err := ioutil.ReadFile("public.key")
	// if err != nil {
	//	t.Fatal(err)
	// }
	//
	// asymmetricKey, err := NewAsymmetricKey(privateKey, publicKey)
	// if err != nil {
	//	t.Fatal(err)
	// }
	////

	// Build a symmetric key
	symmetricKey := encrypt.NewSymmetricKey([]byte("my-secret-key-00"))

	// Build encryption materials which will encrypt uploaded data
	cbcMaterials, err := encrypt.NewCBCSecureMaterials(symmetricKey)
	if err != nil {
		log.Fatalln(err)
	}

	// Get a deciphered data from the server, deciphering is assured by cbcMaterials
	reader, err := s3Client.GetEncryptedObject("my-bucketname", "my-objectname", cbcMaterials)
	if err != nil {
		log.Fatalln(err)
	}
	defer reader.Close()

	// Local file which holds plain data
	localFile, err := os.Create("my-testfile")
	if err != nil {
		log.Fatalln(err)
	}
	defer localFile.Close()

	if _, err := io.Copy(localFile, reader); err != nil {
		log.Fatalln(err)
	}
}
