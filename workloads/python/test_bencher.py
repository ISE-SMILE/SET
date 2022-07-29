#! /usr/env python3

from bencher import PMemory

def cpu_heavy():
    PMemory({
        "operator_size":int(1E4),"itterations":int(1E7),"recursion_depth":20,"threads":4,
    })

def mem_heavy():
    PMemory({
        "operator_size":int(1E7),"itterations":int(1E5),"recursion_depth":10,"threads":4,
    })

def mixed():
    PMemory({
        "operator_size":int(1E6),"itterations":int(1E5),"recursion_depth":20,"threads":4,
    })

if __name__ == "__main__":
    import sys
    args = sys.argv[1:]
    ops = int(args[0])
    itters = int(args[1])
    req = int(args[2])
    PMemory({"operator_size":ops,"itterations":itters,"recursion_depth":req,"threads":8})

    
