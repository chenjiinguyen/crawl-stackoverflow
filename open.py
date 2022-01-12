# Python program to read
# json file

import time
import json
from os import walk
path = './output/'

total_count = 0
start = time.time()
while True:
    
    count = 0
    f = []
    for (dirpath, dirnames, filenames) in walk(path):
        f.extend(filenames)
        break
    for i in f:
        # Opening JSON file
        if ".json" not in i:
            continue
        x = open(path+i, 'r')
        # returns JSON object as
        # a dictionary
        data = json.load(x)
        # Iterating through the json
        # list
        # print(path+i)
        # print(len(data))
        count += len(data);
        # Closing file
        x.close()

    print(count)
    end = time.time()
    if end - start >= 60:
        print("Speed: ", count-total_count , "per minute")
        total_count = count
        start = time.time()
    
    time.sleep(1)


