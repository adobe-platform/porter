FROM ubuntu:16.04

ADD ec2-bootstrap.cmd /
RUN chmod 544 /ec2-bootstrap.cmd

# TODO generate config
CMD /ec2-bootstrap.cmd
