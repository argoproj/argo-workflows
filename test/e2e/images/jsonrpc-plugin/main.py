from flask import Flask, request, jsonify

api = Flask(__name__)
api.config["DEBUG"] = True


@api.route('/', methods=['POST'])
def rpc():
    req = request.json
    result = {
        "init": {"pluginTemplateTypes": ["hello"]},
        "executeNode": {"phase": "Succeeded", "message": "hi"},
        "reconcileNode": {}
    }[req['method']]
    resp = {"id": 0, "jsonrpc": "2.0", "result": result}
    return jsonify(resp)


if __name__ == '__main__':
    api.run(port=12345)
