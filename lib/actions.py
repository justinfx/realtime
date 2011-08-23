#!/usr/bin/env python

import os, sys

THIS = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(THIS)
CONF = os.path.join(ROOT, "etc/supervisord.conf")

SUPERD = os.path.join(ROOT, "supervisord")
SUPERCTL = os.path.join(ROOT, "supervisorctl")

os.environ["PYTHONPATH"] = ':'.join([os.environ.get("PYTHONPATH"), os.path.join(THIS, "supervisor")])


def start():
    cmd = "cd %s && %s -c %s" % (ROOT, SUPERD, CONF)
    ret = os.system(cmd)
    if ret != 0:
        raise Exception('Error starting RealTime Server')
    
def stop(quiet=False):
    cmd = "cd %s && %s -c %s shutdown" % (ROOT, SUPERCTL, CONF)
    if quiet:
        cmd = "%s &> /dev/null" % cmd
    ret = os.system(cmd)
    if ret != 0:
        raise Exception('Error stopping RealTime Server')

def restart(quiet=False):
    try:
        stop(quiet)
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
    cmd = "cd %s && %s -c %s status" % (ROOT, SUPERCTL, CONF)
    if quiet:
        cmd = "%s &> /dev/null" % cmd
    ret = os.system(cmd)
    if ret != 0:
        raise Exception('Error getting status of RealTime Server')
        

