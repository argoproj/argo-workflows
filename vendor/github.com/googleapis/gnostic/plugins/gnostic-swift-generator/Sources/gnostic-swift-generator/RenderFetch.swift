// Copyright 2017 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import Foundation
import Gnostic

extension ServiceRenderer {

    func renderFetch() -> String {
        var code = CodePrinter()
        code.print(header)

        code.print("""
import Foundation
import Dispatch
import KituraNet

// fetch makes a synchronous request using KituraNet's ClientRequest class
// https://github.com/IBM-Swift/Kitura-net/blob/master/Sources/KituraNet/ClientRequest.swift
public func fetch(_ urlRequest: URLRequest) -> (Data?, HTTPURLResponse?, Error?) {
  var data: Data?
  var urlResponse: HTTPURLResponse?
  let error: Error? = nil // make this mutable when we start using it
  let sem = DispatchSemaphore(value: 0)
  guard let method = urlRequest.httpMethod else {
    return (data, urlResponse, error)
  }
  guard let url = urlRequest.url else {
    return (data, urlResponse, error)
  }
  guard let scheme = url.scheme else {
    return (data, urlResponse, error)
  }
  guard let host = url.host else {
    return (data, urlResponse, error)
  }
  guard let port = url.port else {
    return (data, urlResponse, error)
  }
  let options : [ClientRequest.Options] = [
    .method(method),
    .schema(scheme),
    .hostname(host),
    .port(Int16(port)),
    .path(url.path),
    // headers, etc
  ]
  let request = HTTP.request(options) { (response) in
    guard let response = response else {
      sem.signal()
      return
    }
    var responseData = Data()
    do {
      let code = response.httpStatusCode
      try response.readAllData(into: &responseData)
      data = responseData
      urlResponse = HTTPURLResponse(url:url,
                                    statusCode:code.rawValue,
                                    httpVersion:"HTTP/1.1",
                                    headerFields:[:])
      sem.signal()
      return
    } catch {
      sem.signal()
      return
    }
  }
  if let requestData = urlRequest.httpBody {
    request.write(from:requestData)
  }
  request.end() // send the request
  // now wait on the semaphore for a response
  let result = sem.wait(timeout: DispatchTime.distantFuture)
  switch result {
  case .success:
    return (data, urlResponse, error)
  default: // includes .timeout
    return (data, urlResponse, error)
  }
}

// fetch makes an asynchronous request using KituraNet's ClientRequest class
// https://github.com/IBM-Swift/Kitura-net/blob/master/Sources/KituraNet/ClientRequest.swift
public func fetch(_ urlRequest: URLRequest, callback:@escaping (Data?, HTTPURLResponse?, Error?) -> ()) {
  var data: Data?
  var urlResponse: HTTPURLResponse?
  let error: Error? = nil // make this mutable when we start using it
  guard let method = urlRequest.httpMethod else {
    callback (data, urlResponse, error)
    return
  }
  guard let url = urlRequest.url else {
    callback (data, urlResponse, error)
    return
  }
  guard let scheme = url.scheme else {
    callback (data, urlResponse, error)
    return
  }
  guard let host = url.host else {
    callback (data, urlResponse, error)
    return
  }
  guard let port = url.port else {
    callback (data, urlResponse, error)
    return
  }
  let options : [ClientRequest.Options] = [
    .method(method),
    .schema(scheme),
    .hostname(host),
    .port(Int16(port)),
    .path(url.path),
    // headers, etc
  ]
  let request = HTTP.request(options) { (response) in
    guard let response = response else {
      callback (data, urlResponse, nil)
      return
    }
    var responseData = Data()
    do {
      let code = response.httpStatusCode
      try response.readAllData(into: &responseData)
      data = responseData
      urlResponse = HTTPURLResponse(url:url,
                                    statusCode:code.rawValue,
                                    httpVersion:"HTTP/1.1",
                                    headerFields:[:])
      callback (data, urlResponse, nil)
      return
    } catch {
      callback (data, urlResponse, nil)
      return
    }
  }
  if let requestData = urlRequest.httpBody {
    request.write(from:requestData)
  }
  request.end() // send the request
}
""")
        return code.content
    }
}
