#/bin/sh

scp -i ~/.ssh/aws.pem ./"$1" \
    ec2-user@ec2-44-201-218-74.compute-1.amazonaws.com:/home/ec2-user/"$2"
 
