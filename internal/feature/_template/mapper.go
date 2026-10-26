package __name__

func (r Create__Name__Request) toInput() Create__Name__Input {
	return Create__Name__Input{Name: r.Name}
}

func to__Name__Response(x __Name__) __Name__Response {
	return __Name__Response{ID: x.ID, Name: x.Name, CreatedAt: x.CreatedAt.UTC()}
}
