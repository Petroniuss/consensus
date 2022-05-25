import json

from locust import HttpUser, task
from datetime import datetime
from dataclasses import dataclass

KEY_NAME = "/key"


@dataclass
class PutRequest:
    value: str
    previouslyObservedVersion: int

    def to_json(self):
        return json.dumps(self.__dict__)


class HelloWorldUser(HttpUser):
    @task(1)
    def get_value(self):
        self.client.get(KEY_NAME)

    @task(3)
    def get_value_2(self):
        self.client.get("http://raft-example-2:8080/key")

    @task(4)
    def get_value_3(self):
        self.client.get("http://raft-example-3:8080/key")

    @task(2)
    def get_set_value(self):
        response = self.client.get(KEY_NAME)
        prev_version = 0

        # if there's already a value associated with given key.
        if response.ok:
            response_body = response.json()
            prev_version = response_body['version']

        now = datetime.now()
        current_time = now.strftime("%H:%M:%S")

        request = PutRequest(
            value=current_time,
            previouslyObservedVersion=prev_version
        )

        # this succeeds only if the version on the server remains unchanged
        # otherwise it returns { success = false .. }
        response = self.client.put("/key", data=request.to_json())
        response_body = response.json()
