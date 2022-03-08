#/bin/sh

sudo yum update
sudo yum install -y docker
sudo usermod -aG docker $USER
sudo service docker restart
sudo curl -L https://github.com/docker/compose/releases/download/1.23.2/docker-compose-`uname -s`-`uname -m` -o /usr/local/bin/docker-compose
sudo chown $USER /var/run/docker.sock
sudo chmod +x /usr/local/bin/docker-compose
