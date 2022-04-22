# diagram.py
from diagrams import Diagram
from diagrams.aws.compute import EC2
from diagrams.aws.database import RDS
from diagrams.aws.network import ELB

from diagrams.azure.storage import StorageAccounts
from diagrams.k8s.storage import PersistentVolume, PersistentVolumeClaim

import os

def myself() -> str:
        f = os.path.basename(__file__)
        no_ext = '.'.join(f.split('.')[:-1])
        return no_ext

with Diagram(myself(), show=False):
        StorageAccounts("standard") >> PersistentVolume("alice-standard") >> PersistentVolumeClaim("alice-standard")

