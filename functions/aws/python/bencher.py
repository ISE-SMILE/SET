#!/usr/bin/env/python
from multiprocessing import Pool

import boto3
import os
import time
from botocore.client import Config,ClientError

from factclient.fact import Fact
from factclient.io.ConsoleLogging import ConsoleLogging

import random

def init():
	io = ConsoleLogging()
	Fact.boot({"inlcudeEnvironment": False, "io": io, "send_on_update": True})


init()

job_keys = ["prime","memory","IO","idle"]
def Handle(falco,job,context ):
	if not validate(job):
		return {"error":"job not defined correctly"}
	
	Fact.start(context, job)
	if "prime" in job:
		Prime(int(job["prime"]))
	elif "memory" in job:
		Memory(job["memory"])
	elif "IO" in job:
		IO(job["IO"])
	elif "idle" in job:
		Idle(int(job["idle"]))
	else:
		return {"error":"job not defined correctly"}

	return Fact.done(context, "test done", ["no more args"])

def validate(job):
	if isinstance(job, dict):
		for k in job_keys:
			if k in job:
				return True
	
	return False

def randomOpt(left, right):
	a = left[random.randint(0,len(left)-1)]
	b = right[random.randint(0,len(right)-1)]

	c = 0
	if random.uniform(0, 1) < 0.5:
		c = a * b
	else:
		if b > 0:
			c = a / b
		else:
			c = a * b

	if random.uniform(0, 1) < 0.5:
		left[random.randint(0,len(left)-1)] = c
	else:
		right[random.randint(0,len(right)-1)] = c


	return left,right

def compute(i,anchor , left, right):
	if i < anchor:
		compute(i+1, anchor, left, right)
	else:
		randomOpt(left, right)
	
def generateOperatorArray(n):
	data = []
	for i in range(n):
		data.append(random.uniform(0.1, 9.9) * random.randint(1,10000))
	return data

def PMemory(task):
	if not all (k in task for k in (
		"threads","operator_size","itterations","recursion_depth")):
		print("missing key in "+task.keys())
		return
	threads = task['threads']
	ptask = {
		"operator_size":int(task['operator_size']/threads),
		"itterations":int(task['itterations']/threads),
		"recursion_depth":task['recursion_depth']
	}
	with Pool(threads) as p:
		p.map(Memory, [ptask]*threads)

def Memory(task):
	if not all (k in task for k in (
		"operator_size","itterations","recursion_depth")):
		print("missing key in "+task.keys())
		return
	
	left = generateOperatorArray(task["operator_size"])
	right = generateOperatorArray(task["operator_size"])
	for i in range(task["itterations"]):
		compute(0, task["recursion_depth"], left, right)
		if i%100 == 0:
			tmp = left.copy()
			left = right.copy()
			right = tmp

	return left,right
	
def Idle(n):
	time.sleep(n)

def Prime(n):
	"""
	Miller-Rabin primality test.
 
	A return value of False means n is certainly not prime. A return value of
	True means n is very likely a prime.
	"""
	if n!=int(n):
		return False
	n=int(n)
	#Miller-Rabin test for prime
	if n==0 or n==1 or n==4 or n==6 or n==8 or n==9:
		return False
 
	if n==2 or n==3 or n==5 or n==7:
		return True
	s = 0
	d = n-1
	while d%2==0:
		d>>=1
		s+=1
	assert(2**s * d == n-1)
 
	def trial_composite(a):
		if pow(a, d, n) == 1:
			return False
		for i in range(s):
			if pow(a, 2**i * d, n) == n-1:
				return False
		return True  
 
	for i in range(8):#number of trials 
		a = random.randrange(2, n)
		if trial_composite(a):
			return False
 
	return True 

def IO(task):
	if task ==  None :
		return -1, -1, 1
	
	if not all (k in task for k in (
		"endpoint_url","keyId","key","keys","itterations",
		"bucket","read_write","chunk_size")):
		return -1,-1,1
	
	region = "us-east-1"
	if "region" in task:
		region = task["region"]
	
	use_ssl = True
	if "disableSSL" in task:
		use_ssl=task["disableSSL"].lower() == "true"
	
	config=None
	if "S3PathStyle" in task:
		if task["S3PathStyle"].lower() == "true":
			config=Config(signature_version='s3v4'),

	s3 = boto3.client('s3',
		endpoint_url=task["endpoint_url"],
		aws_access_key_id=task["keyId"],
		aws_secret_access_key=task["key"],
		use_ssl=use_ssl,
		config=config,
		region_name=region)

	chunk_size = int(task["chunk_size"])
	objects = {}

	for key in task["keys"]:
		try:
			head = s3.head_object(
				Bucket=task["bucket"],
				Key=key)
			objects[key] = int(head["ContentLength"])
		except Exception as e:
			print("failed to get key %s"%key,e)

	reads = 0 
	writes = 0
	errors = 0

	for i in range(task["itterations"]):
		if random.uniform(0, 1) < task["read_write"]:
			key = random.choice(task["keys"])
			start = random.randint(0,objects[key]-chunk_size)
			rangeString = "bytes=%d-%d"%(start, start+chunk_size)
			try:
				resp = s3.get_object(Bucket=task["bucket"],Key=key,Range=rangeString)
		   
				data = resp['Body'].read()
				reads += len(data)
			except ClientError as e:
				error_code = e.response["Error"]["Code"]
				errors+=1
				print("failed to get %s [%d-%d] - %s"%(key,start,start+chunk_size,error_code))
		else:
			key = "generated_%d.bin"%i
			rnd = os.urandom(chunk_size)
			try:
				resp = s3.put_object(Body=rnd, Bucket=task["bucket"], Key=key)
				writes += chunk_size
			except ClientError as e:
				error_code = e.response["Error"]["Code"]
				errors+=1
				print("failed to write  %d"%(key,error_code))
			
	return reads, writes, errors