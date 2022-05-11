from locust import HttpUser, task
from datetime import datetime

class HelloWorldUser(HttpUser):
    @task(1)
    def get_value(self):
        self.client.get("/key")

    @task(2)
    def set_value(self):
        now = datetime.now()
        current_time = now.strftime("%H:%M:%S")
        self.client.put("/key", data=current_time)
