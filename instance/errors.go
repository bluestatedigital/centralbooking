package instance

type ValidationError struct {
    msg string
}

func (self *ValidationError) Error() string {
    return self.msg
}
