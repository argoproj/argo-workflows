AXOPS_GET_TOOLS_MOCK_RESPONSE = {
    "data": [
        # BitBucket
        {
            "id":"ed6e3c8f-8a6b-11e6-a060-02420af40108",
            "url":"https://api.bitbucket.org",
            "category":"scm",
            "type":"bitbucket",
            "username":"username",
            "password": "password",
            "protocol":"https",
            "all_repos":[
                "https://repo.org/company/demo.git",
                "https://repo.org/company/prod.git"
            ],
            "repos":[
                "https://repo.org/company/prod.git"
            ],
            "use_webhook":False
        },
        # GitHub
        {
            "id":"115a6204-8a9a-11e6-864e-02420af40108",
            "url":"https://api.github.com",
            "category":"scm",
            "type":"github",
            "username":"username",
            "password": "password",
            "protocol":"https",
            "all_repos":[
                "https://repo.org/company/prod.git",
            ],
            "repos":[
                "https://repo.org/company/prod.git"
            ],
            "use_webhook":False
        },
        # Generic Git
        {
            "id":"6fef08d5-8a99-11e6-84fd-02420af40108",
            "url":"https://bitbucket.org/argo/demo.git",
            "category":"scm",
            "type":"git",
            "username":"username",
            "password": "password",
            "protocol":"https",
            "repos":[
                "https://repo.org/company/prod.git"
            ]
        },
        # CodeCommit
        {
            "id": "12490b1d-90ec-4dbb-a9cf-8b9a41caae5e",
            "url": "https://codecommit.us-east-1.amazonaws.com",
            "category": "scm",
            "type": "codecommit",
            "username": "REPLACE_ME_AWS_ACCESS_KEY_ID",
            "password": "REPLACE_ME_AWS_SECRET_ACCESS_KEY",
            "all_repos": [
                "https://git-codecommit.us-east-1.amazonaws.com/v1/repos/goexample",
             ],
            "repos": [
                "https://git-codecommit.us-east-1.amazonaws.com/v1/repos/goexample"
            ]
        },
        # Generic Git (no password)
        {
            "id":"079c64ac-a149-11e6-8a20-0a580af41e55",
            "url":"https://github.com/demo/goexample.git",
            "category":"scm",
            "type":"git",
            "protocol":"https",
            "repos":[
                "https://repo.org/company/prod.git"
            ]
        }
    ]
}
