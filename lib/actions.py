#!/usr/bin/env python

import os, sys, time

THIS = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(THIS)
CONF = os.path.join(ROOT, "etc/supervisord.conf")

SUPERD = os.path.join(ROOT, "bin/realtimed")
SUPERCTL = os.path.join(ROOT, "bin/realtimectl")

ORIG = os.environ.get("PYTHONPATH", '')
P1 = os.path.join(THIS, "supervisor")
P2 = os.path.join(THIS, "meld3")
os.environ["PYTHONPATH"] = ':'.join([P1, P2, ORIG])


def start():
    cmd = "cd %s && %s -c %s" % (ROOT, SUPERD, CONF)
    ret = os.system(cmd)
    if ret != 0:
        raise Exception('Error starting RealTime Server')
    
def stop(quiet=False):
    cmd = "%s -c %s shutdown" % (SUPERCTL, CONF)
    if quiet:
        cmd = "%s &> /dev/null" % cmd
    cmd = 'cd %s && %s' % (ROOT, cmd)
    ret = os.system(cmd)
    if ret != 0:
        raise Exception('Error stopping RealTime Server')

def restart(quiet=False):
    try:
        stop(quiet)
	time.sleep(.5)
    except:
        pass

    while True:
        try:
            status(quiet=True)
        except:
            continue
        break
    
    start()

 
    
def status(quiet=False):
    cmd = "%s -c %s status" % (SUPERCTL, CONF)
    if quiet:
        cmd = "%s &> /dev/null" % cmd
    cmd = 'cd %s && %s' % (ROOT, cmd)
    ret = call(cmd, shell=True)
    if ret != 0:
        raise Exception('Error getting status of RealTime Server')
        


        
