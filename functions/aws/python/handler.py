import json

from bencher import Handle

def bench(event, context):
    body = Handle(None,event,context)

    response = {
        "statusCode": 200,
        "body": json.dumps(body)
    }

    return response