package module

import (
	"github.com/dolotech/leaf/chanrpc"
	"github.com/dolotech/leaf/console"
	"github.com/dolotech/leaf/timer"
	"time"
	"github.com/dolotech/lib/grpool"
	"runtime"
	"github.com/golang/glog"
	"github.com/dolotech/leaf/conf"
)

type Skeleton struct {
	//GoLen              int
	TimerDispatcherLen int
	AsynCallLen        int
	ChanRPCServer      *chanrpc.Server
	//g                  *g.Go
	dispatcher    *timer.Dispatcher
	client        *chanrpc.Client
	server        *chanrpc.Server
	commandServer *chanrpc.Server

	pool *grpool.Pool
}

func (s *Skeleton) Init() {
	/*	if s.GoLen <= 0 {
			s.GoLen = 0
		}*/

	//if s.pool == nil {
	s.pool = grpool.NewPool(runtime.NumCPU()*2, 1024)
	//}
	if s.TimerDispatcherLen <= 0 {
		s.TimerDispatcherLen = 0
	}
	if s.AsynCallLen <= 0 {
		s.AsynCallLen = 0
	}

	//s.g = g.New(s.GoLen)
	s.dispatcher = timer.NewDispatcher(s.TimerDispatcherLen)
	s.client = chanrpc.NewClient(s.AsynCallLen)
	s.server = s.ChanRPCServer

	if s.server == nil {
		s.server = chanrpc.NewServer(0)
	}
	s.commandServer = chanrpc.NewServer(0)
}

func (s *Skeleton) Run(closeSig chan bool) {
	//glog.Errorln("aa")

	for {
		select {
		case <-closeSig:
			s.commandServer.Close()
			s.server.Close()
			//for !s.g.Idle() || !s.client.Idle() {
			for !s.client.Idle() {
				//s.g.Close()
				s.client.Close()
			}
			return
		case ri := <-s.client.ChanAsynRet:
			s.client.Cb(ri)
		case ci := <-s.server.ChanCall:
			s.server.Exec(ci)
		case ci := <-s.commandServer.ChanCall:
			s.commandServer.Exec(ci)
			/*case cb := <-s.g.ChanCb:
				s.g.Cb(cb)*/
		case t := <-s.dispatcher.ChanTimer:
			t.Cb()
		}
	}
}

func (s *Skeleton) AfterFunc(d time.Duration, cb func()) *timer.Timer {
	if s.TimerDispatcherLen == 0 {
		panic("invalid TimerDispatcherLen")
	}

	return s.dispatcher.AfterFunc(d, cb)
}

func (s *Skeleton) CronFunc(cronExpr *timer.CronExpr, cb func()) *timer.Cron {
	if s.TimerDispatcherLen == 0 {
		panic("invalid TimerDispatcherLen")
	}

	return s.dispatcher.CronFunc(cronExpr, cb)
}

func (s *Skeleton) Go(f func(), cb func()) {
	/*if s.GoLen == 0 {
		panic("invalid GoLen")
	}
	s.g.Go(f, cb)*/

	s.pool.JobQueue <- func() {
		defer func() {
			if nil != cb {
				s.pool.JobQueue <- cb
			}
			if r := recover(); r != nil {
				if conf.LenStackBuf > 0 {
					buf := make([]byte, conf.LenStackBuf)
					l := runtime.Stack(buf, false)
					glog.Errorf("%v: %s", r, buf[:l])
				} else {
					glog.Errorf("%v", r)
				}
			}
		}()
		f()
	}
}

/*func (s *Skeleton) NewLinearContext() *g.LinearContext {
	if s.GoLen == 0 {
		panic("invalid GoLen")
	}

	return s.g.NewLinearContext()
}*/

func (s *Skeleton) AsynCall(server *chanrpc.Server, id interface{}, args ...interface{}) {
	if s.AsynCallLen == 0 {
		panic("invalid AsynCallLen")
	}

	s.client.Attach(server)
	s.client.AsynCall(id, args...)
}

func (s *Skeleton) RegisterChanRPC(id interface{}, f interface{}) {
	if s.ChanRPCServer == nil {
		panic("invalid ChanRPCServer")
	}

	s.server.Register(id, f)
}

func (s *Skeleton) RegisterCommand(name string, help string, f interface{}) {
	console.Register(name, help, f, s.commandServer)
}
