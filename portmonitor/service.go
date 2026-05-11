// portmonitor/service.go
// Implementa a interface svc.Handler para rodar como servico Windows.

package main

import (
	"log"

	"golang.org/x/sys/windows/svc"
)

type epsonService struct{}

func (s *epsonService) Execute(args []string, req <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending}

	stop := make(chan struct{})
	go func() {
		log.Println("[servico] Iniciando monitor...")
		runMonitorWithStop(stop)
		log.Println("[servico] Monitor encerrado.")
	}()

	status <- svc.Status{
		State:   svc.Running,
		Accepts: svc.AcceptStop | svc.AcceptShutdown,
	}
	log.Println("[servico] Rodando, aguardando comandos do SCM...")

	for c := range req {
		switch c.Cmd {
		case svc.Stop, svc.Shutdown:
			log.Println("[servico] Parando...")
			status <- svc.Status{State: svc.StopPending}
			close(stop)
			return false, 0
		case svc.Interrogate:
			status <- c.CurrentStatus
		}
	}
	return false, 0
}
