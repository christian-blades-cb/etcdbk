FROM centurylink/ca-certs

MAINTAINER Christian Blades <christian.blades@careerbuilder.com>

ADD etcdbk /

ENTRYPOINT [ "/etcdbk" ]
