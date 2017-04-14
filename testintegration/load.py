from locust import HttpLocust, TaskSet
import random


def empty(l):
    l.client.request('GET', '/empty')


def rand_latency(l):
    ms = '{}ms'.format(random.randrange(1, 1000))
    l.client.request('GET', '/load', None, False, headers={'X-Response-Time': ms})


def no_keep_alive(l):
    l.client.request('GET', '/hello', None, False, headers={'Connection': 'close'})


class UserBehavior(TaskSet):
    # tasks = {rand_latency:1, no_keep_alive:1}
    tasks = [empty]


class WebsiteUser(HttpLocust):
    task_set = UserBehavior
